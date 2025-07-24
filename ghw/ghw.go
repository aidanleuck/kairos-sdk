package ghw

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kairos-io/kairos-sdk/types"
)

const (
	sectorSize = 512
	UNKNOWN    = "unknown"
)

type Paths struct {
	SysBlock    string
	RunUdevData string
	ProcMounts  string
}

func NewPaths(withOptionalPrefix string) *Paths {
	p := &Paths{
		SysBlock:    "/sys/block/",
		RunUdevData: "/run/udev/data",
		ProcMounts:  "/proc/mounts",
	}

	// Allow overriding the paths via env var. It has precedence over anything
	val, exists := os.LookupEnv("GHW_CHROOT")
	if exists {
		val = strings.TrimSuffix(val, "/")
		p.SysBlock = fmt.Sprintf("%s%s", val, p.SysBlock)
		p.RunUdevData = fmt.Sprintf("%s%s", val, p.RunUdevData)
		p.ProcMounts = fmt.Sprintf("%s%s", val, p.ProcMounts)
		return p
	}

	if withOptionalPrefix != "" {
		withOptionalPrefix = strings.TrimSuffix(withOptionalPrefix, "/")
		p.SysBlock = fmt.Sprintf("%s%s", withOptionalPrefix, p.SysBlock)
		p.RunUdevData = fmt.Sprintf("%s%s", withOptionalPrefix, p.RunUdevData)
		p.ProcMounts = fmt.Sprintf("%s%s", withOptionalPrefix, p.ProcMounts)
	}
	return p
}

func isMultipathDevice(entry os.DirEntry) bool {
	return strings.HasPrefix(entry.Name(), "dm-")
}

func GetDisks(paths *Paths, logger *types.KairosLogger) []*types.Disk {
	if logger == nil {
		newLogger := types.NewKairosLogger("ghw", "info", false)
		logger = &newLogger
	}
	disks := make([]*types.Disk, 0)
	logger.Logger.Debug().Str("path", paths.SysBlock).Msg("Scanning for disks")
	files, err := os.ReadDir(paths.SysBlock)
	if err != nil {
		return nil
	}
	for _, file := range files {
		var partitionHandler PartitionHandler;
		logger.Logger.Debug().Str("file", file.Name()).Msg("Reading file")
		dname := file.Name()
		size := diskSizeBytes(paths, dname, logger)

		// Skip entries that are multipath partitions
		// we will handle them when we parse this disks partitions
		if isMultipathPartition(file, paths) {
			logger.Logger.Debug().Str("file", dname).Msg("Skipping multipath partition")
			continue
		}

		if strings.HasPrefix(dname, "loop") && size == 0 {
			// We don't care about unused loop devices...
			continue
		}
		d := &types.Disk{
			Name:      dname,
			SizeBytes: size,
			UUID:      diskUUID(paths, dname, "", logger),
		}

		if(isMultipathDevice(file)) {
			partitionHandler = NewMultipathPartitionHandler(dname)
		} else {
			partitionHandler = NewDiskPartitionHandler(dname)
		}
		

		parts := partitionHandler.GetPartitions(paths, logger)
		d.Partitions = parts

		disks = append(disks, d)
	}

	return disks
}

func isMultipathPartition(entry os.DirEntry, paths *Paths) bool {
    // Must be a dm device to be a multipath partition
    if !strings.HasPrefix(entry.Name(), "dm-") {
        return false
    }
    
    // Check for dm/uuid file existence
    uuidPath := filepath.Join(paths.SysBlock, entry.Name(), "dm/uuid")
    uuidBytes, err := os.ReadFile(uuidPath)
    if err != nil {
        return false
    }

    uuid := strings.TrimSpace(string(uuidBytes))
    
    // Multipath partitions typically have UUIDs indicating they are partitions
    // Common patterns: "part1-mpath-xxx", "mpath-xxx-part1", etc.
    return strings.HasPrefix(uuid, "part") || 
           strings.Contains(uuid, "-part") || 
           (strings.Contains(uuid, "mpath") && strings.Contains(uuid, "part"))
}

func diskSizeBytes(paths *Paths, disk string, logger *types.KairosLogger) uint64 {
	// We can find the number of 512-byte sectors by examining the contents of
	// /sys/block/$DEVICE/size and calculate the physical bytes accordingly.
	path := filepath.Join(paths.SysBlock, disk, "size")
	logger.Logger.Debug().Str("path", path).Msg("Reading disk size")
	contents, err := os.ReadFile(path)
	if err != nil {
		logger.Logger.Error().Str("path", path).Err(err).Msg("Failed to read file")
		return 0
	}
	size, err := strconv.ParseUint(strings.TrimSpace(string(contents)), 10, 64)
	if err != nil {
		logger.Logger.Error().Str("path", path).Err(err).Str("content", string(contents)).Msg("Failed to parse size")
		return 0
	}
	logger.Logger.Trace().Uint64("size", size*sectorSize).Msg("Got disk size")
	return size * sectorSize
}

package ghw

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kairos-io/kairos-sdk/types"
)

type MultipathPartitionHandler struct {
	DiskName string
}

func NewMultipathPartitionHandler(diskName string) *MultipathPartitionHandler {
	return &MultipathPartitionHandler{DiskName: diskName}
}

var _ PartitionHandler = &MultipathPartitionHandler{}

func (m *MultipathPartitionHandler) GetPartitions(paths *Paths, logger *types.KairosLogger) types.PartitionList {
	out := make(types.PartitionList, 0)

	// For multipath devices, partitions appear as holders of the parent device
	holdersPath := filepath.Join(paths.SysBlock, m.DiskName, "holders")
	logger.Logger.Debug().Str("path", holdersPath).Msg("Reading multipath holders")

	holders, err := os.ReadDir(holdersPath)
	if err != nil {
		logger.Logger.Error().Err(err).Msg("failed to read holders directory")
		return out
	}

	for _, holder := range holders {
		partName := holder.Name()

		// Only consider dm- devices as potential multipath partitions
		if !isMultipathDevice(holder) {
			continue
		}
		
		// Verify this holder is actually a multipath partition
		// We need to read the directory to get the DirEntry for the partition
		partParentDir := filepath.Join(paths.SysBlock)
		partFiles, err := os.ReadDir(partParentDir)
		if err != nil {
			logger.Logger.Debug().Str("partition", partName).Err(err).Msg("Could not read parent directory")
			continue
		}
		
		var partEntry os.DirEntry
		found := false
		for _, pf := range partFiles {
			if pf.Name() == partName {
				partEntry = pf
				found = true
				break
			}
		}
		
		if !found {
			logger.Logger.Debug().Str("partition", partName).Msg("Could not find DirEntry for partition")
			continue
		}
		
		if !isMultipathPartition(partEntry, paths) {
			logger.Logger.Debug().Str("partition", partName).Msg("Holder is not a multipath partition")
			continue
		}

		logger.Logger.Debug().Str("partition", partName).Msg("Found multipath partition")

		// For multipath partitions, we need to get size directly from the partition device
		// since it's a top-level entry in /sys/block, not nested under the parent
		size := diskSizeBytes(paths, partName, logger)
		mp, pt := partitionInfo(paths, partName, logger)

		// For multipath partitions, we need to get udev info directly from the partition
		// Get device number for the partition
		devPath := filepath.Join(paths.SysBlock, partName, "dev")
		devNoBytes, err := os.ReadFile(devPath)
		if err != nil {
			logger.Logger.Error().Err(err).Str("path", devPath).Msg("Failed to read device number")
			continue
		}

		devNo := strings.TrimSpace(string(devNoBytes))
		udevInfo, err := UdevInfo(paths, devNo, logger)
		if err != nil {
			logger.Logger.Error().Err(err).Str("devNo", devNo).Msg("Failed to get udev info")
			continue
		}

		// Extract UUID from udev info
		du := UNKNOWN
		if val, ok := udevInfo["ID_PART_ENTRY_UUID"]; ok {
			du = val
		}

		// Extract filesystem label from udev info
		fsLabel := UNKNOWN
		if val, ok := udevInfo["ID_FS_LABEL"]; ok {
			fsLabel = val
		}

		// Get filesystem type if not from mount info
		if pt == "" {
			if val, ok := udevInfo["ID_FS_TYPE"]; ok {
				pt = val
			} else {
				pt = UNKNOWN
			}
		}

		p := &types.Partition{
			Name:            partName,
			Size:            uint(size / (1024 * 1024)),
			MountPoint:      mp,
			UUID:            du,
			FilesystemLabel: fsLabel,
			FS:              pt,
			Path:            filepath.Join("/dev", partName),
			Disk:            filepath.Join("/dev", m.DiskName),
		}
		out = append(out, p)
	}

	return out
}
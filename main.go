package main

import (
	"os"

	"github.com/kairos-io/kairos-sdk/ghw"
	"github.com/kairos-io/kairos-sdk/types"
	"github.com/rs/zerolog"
)

// func checkNotNil(parts *v1.ElementalPartitions) error {
// 	if parts == nil {
// 		return fmt.Errorf("ElementalPartitions is nil")
// 	}

// 	if parts.EFI == nil && parts.OEM == nil && parts.Persistent == nil && parts.Recovery == nil && parts.State == nil {
// 		return fmt.Errorf("All partition types are nil")
// 	}

// 	return nil
// }

func printPartition(partition *types.Partition, logger *zerolog.Logger) {
	logger.Info().Msgf("    Name: %s", partition.Name)
	logger.Info().Msgf("    Path: %s", partition.Path)
	logger.Info().Msgf("    Disk: %s", partition.Disk)
	logger.Info().Msgf("    Label: %s", partition.FilesystemLabel)
	logger.Info().Msgf("    Mount Point: %s", partition.MountPoint)
	logger.Info().Msg("")
}

func printElementalPartition(partType string, partition *types.Partition, logger *zerolog.Logger) {
	if partition == nil {
		logger.Warn().Msgf("  %s: Not found", partType)
		return
	}

	logger.Info().Msgf("  %s:", partType)
	printPartition(partition, logger)

}

func getAllPartitions(logger *types.KairosLogger) (types.PartitionList, error) {
	var parts []*types.Partition

	for _, d := range ghw.GetDisks(ghw.NewPaths(""), logger) {
		for _, part := range d.Partitions {
			parts = append(parts, part)
		}
	}
	return parts, nil
}

func main() {
	logger := zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	kairosLogger := types.KairosLogger{
		Logger: logger,
	}

	logger.Info().Msg("=== Kairos Disk Discovery Tool ===")
	logger.Info().Msg("")

	// Emulate the behavior of GetDisks in Kairos so we can see what disks it is discovering
	//https://github.com/kairos-io/kairos-agent/blob/2e1b98cf1abc3d98a6d50fb10be5e80a63dff185/pkg/utils/partitions/getpartitions.go#L36
	logger.Info().Msg("🔍 Discovering available disks...")
	disks := ghw.GetDisks(ghw.NewPaths(""), &kairosLogger)
	if len(disks) == 0 {
		logger.Error().Msg("❌ No disks found")
		os.Exit(1)
	}

	logger.Info().Msgf("✅ Successfully discovered %d disk(s)", len(disks))
	logger.Info().Msg("")
	logger.Info().Msg("📀 DISK INFORMATION:")
	logger.Info().Msg("────────────────────")

	for i, disk := range disks {
		sizeGB := float64(disk.SizeBytes) / (1024 * 1024 * 1024)
		logger.Info().Msgf("  [%d] Name: %s", i+1, disk.Name)
		logger.Info().Msgf("      Size: %.2f GB (%d bytes)", sizeGB, disk.SizeBytes)
		logger.Info().Msg("")
	}

	logger.Info().Msg("🔍 Scanning for partitions...")
	// Retrieve partitions using the GetAllPartitions function
	// This will return all partitions across all disks
	partitions, err := getAllPartitions(&kairosLogger)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Failed to retrieve partitions")
		os.Exit(1)
	}

	if len(partitions) == 0 {
		logger.Error().Msg("❌ No partitions found")
		os.Exit(1)
	}

	logger.Info().Msgf("✅ Successfully found %d partition(s)", len(partitions))
	logger.Info().Msg("")
	logger.Info().Msg("💾 PARTITION INFORMATION:")
	logger.Info().Msg("─────────────────────────")

	// Iterate over each partition and print its details
	for _, partition := range partitions {
		printPartition(partition, &logger)
	}

	// Filters on the partitions and separates them into their respective categories
	// EFI, BIOS, etc.

	// logger.Info().Msg("🔄 Categorizing partitions by type...")
	// partitionsList := v1.NewElementalPartitionsFromList(partitions)
	// if err := checkNotNil(&partitionsList); err != nil {
	// 	logger.Error().Err(err).Msg("❌ ElementalPartitions is not properly initialized")
	// 	os.Exit(1)
	// }

	// logger.Info().Msg("✅ Partitions successfully categorized")
	// logger.Info().Msg("")
	// logger.Info().Msg("📂 PARTITION CATEGORIES:")
	// logger.Info().Msg("────────────────────────")

	// // Partitions are printed out by type after being separated into categories
	// printElementalPartition("EFI", partitionsList.EFI, &logger)
	// printElementalPartition("OEM", partitionsList.OEM, &logger)
	// printElementalPartition("Persistent", partitionsList.Persistent, &logger)
	// printElementalPartition("Recovery", partitionsList.Recovery, &logger)
	// printElementalPartition("State", partitionsList.State, &logger)

	// logger.Info().Msg("")
	// logger.Info().Msg("🗂️ PARTITIONS BY MOUNT POINT:")
	// logger.Info().Msg("──────────────────────────────")

	// // Retrieve partitions by mount point
	// mountPointPartitions := partitionsList.PartitionsByMountPoint(false)
	// if len(mountPointPartitions) == 0 {
	// 	logger.Warn().Msg("⚠️  No partitions found with mount points")
	// } else {
	// 	logger.Info().Msgf("📍 Found %d partition(s) with mount points", len(mountPointPartitions))
	// 	logger.Info().Msg("")

	// 	// Print out the partitions found by mount point

	// 	for _, partition := range mountPointPartitions {
	// 		printPartition(partition, &logger)
	// 	}
	// }

	logger.Info().Msg("🎉 Disk and partition analysis completed successfully!")
}

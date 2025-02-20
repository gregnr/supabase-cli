package stop

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/spf13/afero"
	"github.com/supabase/cli/internal/utils"
)

func Run(ctx context.Context, backup bool, fsys afero.Fs) error {
	// Sanity checks.
	if err := utils.LoadConfigFS(fsys); err != nil {
		return err
	}

	// Stop all services
	if err := stop(ctx, backup); err != nil {
		return err
	}

	fmt.Println("Stopped " + utils.Aqua("supabase") + " local development setup.")
	return nil
}

func stop(ctx context.Context, backup bool) error {
	args := filters.NewArgs(
		filters.Arg("label", "com.supabase.cli.project="+utils.Config.ProjectId),
	)
	containers, err := utils.Docker.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return err
	}
	// Gracefully shutdown containers
	var ids []string
	for _, c := range containers {
		if c.State == "running" {
			ids = append(ids, c.ID)
		}
	}
	utils.WaitAll(ids, utils.DockerStop)
	if _, err := utils.Docker.ContainersPrune(ctx, args); err != nil {
		return err
	}
	// Remove named volumes
	if !backup {
		// TODO: label named volumes to use VolumesPrune for branch support
		volumes := []string{utils.DbId, utils.StorageId}
		utils.WaitAll(volumes, func(name string) {
			if err := utils.Docker.VolumeRemove(ctx, name, true); err != nil {
				fmt.Fprintln(os.Stderr, "failed to remove volume:", name, err)
			}
		})
	}
	// Remove networks.
	_, err = utils.Docker.NetworksPrune(ctx, args)
	return err
}

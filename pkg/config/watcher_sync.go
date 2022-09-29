package config

import (
	"context"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	watcherPollInterval = 200 * time.Millisecond
)

// WaitForWatcherSync delays startup until ConfigWatcher synchronizes at least one configuration file
func WaitForWatcherSync(ctx context.Context, log logrus.FieldLogger, cfg CfgWatcher) error {
	if cfg.InitialSyncTimeout.Milliseconds() == 0 {
		log.Info("Skipping waiting for Config Watcher sync...")
		return nil
	}

	log.Infof("Waiting for synchronized files in directory %q with timeout %s...", cfg.TmpDir, cfg.InitialSyncTimeout)
	err := wait.PollWithContext(ctx, watcherPollInterval, cfg.InitialSyncTimeout, func(ctx context.Context) (done bool, err error) {
		files, err := os.ReadDir(cfg.TmpDir)
		if err != nil {
			return false, err
		}

		for _, file := range files {
			if file.IsDir() {
				// skip subdirectories
				continue
			}

			log.Infof("File %q detected. Finishing polling...", file.Name())
			return true, nil
		}

		return false, nil
	})

	return err
}

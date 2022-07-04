package controller

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/notify"
)

// ConfigWatcher watches for the config file changes and exits the app.
// TODO: It keeps the previous behavior for now, but it should hot-reload the configuration files without needing a restart.
//  Also, it should watch both files.
type ConfigWatcher struct {
	log         logrus.FieldLogger
	configPath  string
	clusterName string
	notifiers   []notify.Notifier
}

// NewConfigWatcher returns new ConfigWatcher instance.
func NewConfigWatcher(log logrus.FieldLogger, configPath string, clusterName string, notifiers []notify.Notifier) *ConfigWatcher {
	return &ConfigWatcher{log: log, configPath: configPath, clusterName: clusterName, notifiers: notifiers}
}

// Do starts watching the configuration file
func (w *ConfigWatcher) Do(ctx context.Context, cancelFunc context.CancelFunc) (err error) {
	configFile := filepath.Join(w.configPath, config.ResourceConfigFileName)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("while creating file watcher: %w", err)
	}
	defer func() {
		deferredErr := watcher.Close()
		if deferredErr != nil {
			err = multierror.Append(err, deferredErr)
		}
	}()

	ctx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				w.log.Info("Shutdown requested. Finishing...")
				return nil
			case _, ok := <-watcher.Events:
				if !ok {
					return fmt.Errorf("unexpected file watch end")
				}

				w.log.Infof("Config file %q is updated. Sending last message before exit...", configFile)
				err := sendMessageToNotifiers(ctx, w.notifiers, fmt.Sprintf(configUpdateMsg, w.clusterName))
				if err != nil {
					wrappedErr := fmt.Errorf("while sending message to notifiers: %w", err)
					//do not exit yet, cancel the context first
					cancelFunc()
					return wrappedErr
				}

				w.log.Infof("Cancelling the context...")
				cancelFunc()
				return nil
			case err, ok := <-watcher.Errors:
				if !ok {
					return fmt.Errorf("unexpected file watch end")
				}
				return fmt.Errorf("while reading events for config file %q: %w", configFile, err)
			}
		}
	})

	w.log.Infof("Registering watcher on config file %q", configFile)
	err = watcher.Add(configFile)
	if err != nil {
		return fmt.Errorf("while registering watch on config file %q: %w", configFile, err)
	}

	return errGroup.Wait()
}

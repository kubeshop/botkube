package controller

import (
	"context"
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"github.com/kubeshop/botkube/pkg/multierror"
)

// ConfigWatcher watches for the config file changes and exits the app.
// TODO: It keeps the previous behavior for now, but it should hot-reload the configuration files without needing a restart.
type ConfigWatcher struct {
	log         logrus.FieldLogger
	configPaths []string
	clusterName string
	notifiers   []Notifier
}

// NewConfigWatcher returns new ConfigWatcher instance.
func NewConfigWatcher(log logrus.FieldLogger, configPaths []string, clusterName string, notifiers []Notifier) *ConfigWatcher {
	return &ConfigWatcher{
		log:         log,
		configPaths: configPaths,
		clusterName: clusterName,
		notifiers:   notifiers,
	}
}

// Do starts watching the configuration file
func (w *ConfigWatcher) Do(ctx context.Context, cancelFunc context.CancelFunc) (err error) {
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

	log := w.log.WithField("configPaths", w.configPaths)

	errGroup, _ := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				log.Info("Shutdown requested. Finishing...")
				return nil
			case ev, ok := <-watcher.Events:
				if !ok {
					return fmt.Errorf("unexpected file watch end")
				}

				currentLogg := log.WithField("event", ev.String())

				currentLogg.Info("Config updated. Sending last message before exit...")
				err := sendMessageToNotifiers(ctx, w.notifiers, fmt.Sprintf(configUpdateMsg, w.clusterName))
				if err != nil {
					wrappedErr := fmt.Errorf("while sending message to notifiers: %w", err)
					//do not exit yet, cancel the context first
					cancelFunc()
					return wrappedErr
				}

				currentLogg.Infof("Cancelling the context...")
				cancelFunc()
				return nil
			case err, ok := <-watcher.Errors:
				if !ok {
					return fmt.Errorf("unexpected file watch end")
				}
				return fmt.Errorf("while reading events for config files: %w", err)
			}
		}
	})

	log.Infof("Registering watcher on config files")
	for _, path := range w.configPaths {
		err = watcher.Add(path)
		if err != nil {
			return fmt.Errorf("while registering watch on config file %q: %w", path, err)
		}
	}
	return errGroup.Wait()
}

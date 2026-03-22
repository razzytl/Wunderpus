package config

import (
	"log/slog"

	"github.com/fsnotify/fsnotify"
)

// Watch starts a file watcher on the given config path and calls onChange
// whenever the file is modified. The callback receives the reloaded config.
func Watch(path string, onChange func(*Config)) (*Config, func(), error) {
	// Initial load
	cfg, err := Load(path)
	if err != nil {
		return nil, nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, nil, err
	}

	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return nil, nil, err
	}

	stop := make(chan struct{})
	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					slog.Info("config: file changed, reloading", "path", path)
					newCfg, err := Load(path)
					if err != nil {
						slog.Error("config: reload failed", "error", err)
						continue
					}
					onChange(newCfg)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("config: watcher error", "error", err)
			case <-stop:
				return
			}
		}
	}()

	cleanup := func() {
		close(stop)
	}

	return cfg, cleanup, nil
}

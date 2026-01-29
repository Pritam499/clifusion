// Copyright 2013-2023 The Cobra Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cobra

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	"github.com/fsnotify/fsnotify"
	lua "github.com/yuin/gopher-lua"
)

var pluginWatcherStarted bool
var pluginMutex sync.Mutex
var globalRootCmd *Command

type Plugin interface {
	Init(rootCmd *Command) error
}

func LoadPlugins(rootCmd *Command) error {
	pluginMutex.Lock()
	defer pluginMutex.Unlock()

	globalRootCmd = rootCmd

	if !pluginWatcherStarted {
		go startPluginWatcher()
		pluginWatcherStarted = true
	}

	pluginDir := filepath.Join(os.Getenv("HOME"), ".cobra", "plugins")
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return nil // No plugins
	}

	return filepath.Walk(pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		switch ext {
		case ".so":
			return loadGoPlugin(path, rootCmd)
		case ".lua":
			return loadLuaPlugin(path, rootCmd)
		}
		return nil
	})
}

func loadGoPlugin(path string, rootCmd *Command) error {
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}
	sym, err := p.Lookup("Init")
	if err != nil {
		return err
	}
	initFunc, ok := sym.(func(*Command) error)
	if !ok {
		return fmt.Errorf("plugin %s has invalid Init function", path)
	}
	return initFunc(rootCmd)
}

func loadLuaPlugin(path string, rootCmd *Command) error {
	L := lua.NewState()
	defer L.Close()

	// Expose functions to Lua
	L.SetGlobal("add_command", L.NewFunction(func(L *lua.LState) int {
		name := L.ToString(1)
		short := L.ToString(2)
		run := L.ToFunction(3)

		cmd := &Command{
			Use:   name,
			Short: short,
			RunE: func(cmd *Command, args []string) error {
				L.Push(run)
				for _, arg := range args {
					L.Push(lua.LString(arg))
				}
				return L.PCall(len(args), 0, nil)
			},
		}
		rootCmd.AddCommand(cmd)
		return 0
	}))

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return L.DoString(string(content))
}

func startPluginWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	defer watcher.Close()

	pluginDir := filepath.Join(os.Getenv("HOME"), ".cobra", "plugins")
	os.MkdirAll(pluginDir, 0755)
	watcher.Add(pluginDir)

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove) != 0 {
				pluginMutex.Lock()
				if globalRootCmd != nil {
					// Clear existing plugin commands? Hard, for now reload all
					LoadPlugins(globalRootCmd)
				}
				pluginMutex.Unlock()
			}
		case err := <-watcher.Errors:
			fmt.Println("Plugin watcher error:", err)
		}
	}
}

func init() {
	// Load plugins on package init, but since rootCmd not available, perhaps call in ExecuteC
}
package utils

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"reflect"
	"sync"
)

// ConfigLoader 配置加载器
type ConfigLoader[T any] struct {
	config     T
	configPath string
	mutex      sync.RWMutex
	watchers   []ConfigWatcher
}

// ConfigWatcher 配置变更监听器
type ConfigWatcher interface {
	OnConfigChange(interface{})
}

// NewConfigLoader 创建新的配置加载器
func NewConfigLoader[T any](defaultConfig T) *ConfigLoader[T] {
	return &ConfigLoader[T]{
		config:   defaultConfig,
		watchers: make([]ConfigWatcher, 0),
	}
}

// LoadYamlConfigFile 加载YAML配置文件
func LoadYamlConfigFile[T any](paths ...string) (cfg T, err error) {
	loader := NewConfigLoader(cfg)
	return loader.LoadFromPaths(paths...)
}

// LoadFromPaths 从多个路径加载配置
func (l *ConfigLoader[T]) LoadFromPaths(paths ...string) (T, error) {
	var cfg T
	if reflect.TypeOf(cfg).Kind() != reflect.Struct {
		return cfg, errors.New("配置类型必须是结构体")
	}

	var lastErr error
	for _, path := range paths {
		if err := l.loadFromPath(path); err != nil {
			GetLogger().Warnf("从路径 %s 加载配置失败: %v", path, err)
			lastErr = err
			continue
		}
		l.configPath = path
		l.notifyWatchers()
		return l.GetConfig(), nil
	}

	if lastErr != nil {
		return cfg, fmt.Errorf("加载配置文件失败: %v", lastErr)
	}
	return cfg, errors.New("未找到有效的配置文件")
}

// loadFromPath 从单个路径加载配置
func (l *ConfigLoader[T]) loadFromPath(path string) error {
	// 验证文件扩展名
	if ext := filepath.Ext(path); ext != ".yml" && ext != ".yaml" {
		return fmt.Errorf("不支持的文件类型: %s", ext)
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("配置文件不存在: %s", path)
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	if err = yaml.Unmarshal(file, &l.config); err != nil {
		return fmt.Errorf("解析YAML失败: %v", err)
	}

	GetLogger().Infof("成功加载配置文件: %s", path)
	return nil
}

// GetConfig 获取当前配置
func (l *ConfigLoader[T]) GetConfig() T {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.config
}

// AddWatcher 添加配置变更监听器
func (l *ConfigLoader[T]) AddWatcher(watcher ConfigWatcher) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.watchers = append(l.watchers, watcher)
}

// RemoveWatcher 移除配置变更监听器
func (l *ConfigLoader[T]) RemoveWatcher(watcher ConfigWatcher) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for i, w := range l.watchers {
		if w == watcher {
			l.watchers = append(l.watchers[:i], l.watchers[i+1:]...)
			break
		}
	}
}

// notifyWatchers 通知所有监听器配置已变更
func (l *ConfigLoader[T]) notifyWatchers() {
	for _, watcher := range l.watchers {
		watcher.OnConfigChange(l.config)
	}
}

// SaveConfig 保存配置到文件
func (l *ConfigLoader[T]) SaveConfig() error {
	if l.configPath == "" {
		return errors.New("未设置配置文件路径")
	}

	l.mutex.RLock()
	data, err := yaml.Marshal(l.config)
	l.mutex.RUnlock()

	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err = os.WriteFile(l.configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	GetLogger().Infof("成功保存配置到: %s", l.configPath)
	return nil
}

// UpdateConfig 更新配置
func (l *ConfigLoader[T]) UpdateConfig(updater func(T) T) error {
	l.mutex.Lock()
	l.config = updater(l.config)
	l.mutex.Unlock()

	l.notifyWatchers()
	return l.SaveConfig()
}

// Validate 验证配置
func (l *ConfigLoader[T]) Validate(validator func(T) error) error {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return validator(l.config)
}

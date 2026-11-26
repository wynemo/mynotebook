//go:build !darwin && !linux && !windows

package notebook

import "errors"

func Move(src, dst string) error { return errors.New("此平台暂不支持安全移动") }

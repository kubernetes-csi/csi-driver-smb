package utils

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/windows"
	"k8s.io/klog/v2"
)

const (
	MaxPathLengthWindows = 260

	// LongPathPrefix is the prefix of Windows long path
	LongPathPrefix = `\\?\`
)

func EnsureLongPath(path string) string {
	if !strings.HasPrefix(path, LongPathPrefix) {
		path = LongPathPrefix + path
	}
	return path
}

func IsPathValid(path string) (bool, error) {
	pathString, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, fmt.Errorf("invalid path: %w", err)
	}

	attrs, err := windows.GetFileAttributes(pathString)
	if err != nil {
		if errors.Is(err, windows.ERROR_PATH_NOT_FOUND) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) || errors.Is(err, windows.ERROR_INVALID_NAME) {
			return false, nil
		}

		// GetFileAttribute returns user or password incorrect for a disconnected SMB connection after the password is changed
		return false, fmt.Errorf("failed to get path %s attribute: %w", path, err)
	}

	klog.V(6).Infof("Path %s attribute: %O", path, attrs)
	return attrs != windows.INVALID_FILE_ATTRIBUTES, nil
}

// IsMountedFolder checks whether the `path` is a mounted folder.
func IsMountedFolder(path string) (bool, error) {
	// https://learn.microsoft.com/en-us/windows/win32/fileio/determining-whether-a-directory-is-a-volume-mount-point
	utf16Path, _ := windows.UTF16PtrFromString(path)
	attrs, err := windows.GetFileAttributes(utf16Path)
	if err != nil {
		return false, err
	}

	if (attrs & windows.FILE_ATTRIBUTE_REPARSE_POINT) == 0 {
		return false, nil
	}

	var findData windows.Win32finddata
	findHandle, err := windows.FindFirstFile(utf16Path, &findData)
	if err != nil && !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return false, err
	}

	for err == nil {
		if findData.Reserved0&windows.IO_REPARSE_TAG_MOUNT_POINT != 0 {
			return true, nil
		}

		err = windows.FindNextFile(findHandle, &findData)
		if err != nil && !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			return false, err
		}
	}

	return false, nil
}

func IsPathSymlink(path string) (bool, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	// for windows NTFS, check if the path is symlink instead of directory.
	isSymlink := fi.Mode()&os.ModeSymlink != 0 || fi.Mode()&os.ModeIrregular != 0
	return isSymlink, nil
}

func CreateSymlink(link, target string, isDir bool) error {
	linkPtr, err := windows.UTF16PtrFromString(link)
	if err != nil {
		return err
	}
	targetPtr, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}

	var flags uint32
	if isDir {
		flags = windows.SYMBOLIC_LINK_FLAG_DIRECTORY
	}

	err = windows.CreateSymbolicLink(
		linkPtr,
		targetPtr,
		flags,
	)
	return err
}

// PathExists checks whether the given `path` exists.
func PathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

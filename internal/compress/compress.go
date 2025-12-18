package compress

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"
)

// CompressDir 压缩整个目录为 tar 格式（支持 zstd、gzip 或不压缩）
// sourceDir: 源目录路径
// outputFile: 输出文件路径
// excludePatterns: 排除模式列表，支持 glob 模式（如 *.log, node_modules, .git）
// compressMethod: 压缩方式 (zstd/gzip/none)，默认为 zstd
func CompressDir(sourceDir, outputFile string, excludePatterns []string, compressMethod string) error {
	// 验证源目录是否存在
	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("源目录不存在: %v", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("源路径不是目录: %s", sourceDir)
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 根据压缩方式创建压缩 writer
	var compressWriter io.WriteCloser
	switch compressMethod {
	case "gzip":
		compressWriter = gzip.NewWriter(outFile)
	case "zstd", "":
		// 默认为 zstd
		zstdWriter, err := zstd.NewWriter(outFile)
		if err != nil {
			return fmt.Errorf("创建 zstd writer 失败: %v", err)
		}
		compressWriter = zstdWriter
	case "none":
		// 不压缩，直接使用文件
		compressWriter = &nopCloser{Writer: outFile}
	default:
		return fmt.Errorf("不支持的压缩方式: %s，支持的方式: zstd, gzip, none", compressMethod)
	}
	defer compressWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(compressWriter)
	defer tarWriter.Close()

	// 获取源目录的绝对路径
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	// 遍历目录并写入文件
	err = filepath.Walk(absSourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(absSourceDir, path)
		if err != nil {
			return fmt.Errorf("获取相对路径失败: %v", err)
		}

		// 检查是否应该排除（同时检查绝对路径和相对路径）
		if shouldExclude(path, relPath, absSourceDir, excludePatterns) {
			if info.IsDir() {
				return filepath.SkipDir // 跳过整个目录
			}
			return nil // 跳过文件
		}

		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("创建tar header失败: %v", err)
		}
		header.Name = relPath

		// 写入header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("写入tar header失败: %v", err)
		}

		// 如果是目录或符号链接，不需要写入内容
		if info.Mode().IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}
		defer file.Close()

		// 复制文件内容
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return fmt.Errorf("复制文件内容失败: %v", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("压缩目录失败: %v", err)
	}

	return nil
}

// shouldExclude 检查文件或目录是否应该被排除
// absPath: 文件的绝对路径
// relPath: 文件相对于源目录的相对路径
// sourceDir: 源目录的绝对路径
// excludePatterns: 排除模式列表
func shouldExclude(absPath, relPath, sourceDir string, excludePatterns []string) bool {
	if len(excludePatterns) == 0 {
		return false
	}

	// 将路径统一使用正斜杠（用于匹配）
	normalizedRelPath := strings.ReplaceAll(relPath, string(filepath.Separator), "/")
	normalizedAbsPath := strings.ReplaceAll(absPath, string(filepath.Separator), "/")

	for _, pattern := range excludePatterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// 规范化模式路径
		normalizedPattern := strings.ReplaceAll(pattern, string(filepath.Separator), "/")
		// 移除模式末尾的斜杠（如果有），以便匹配目录及其内容
		normalizedPattern = strings.TrimSuffix(normalizedPattern, "/")

		// 如果模式是绝对路径，检查是否匹配绝对路径
		if filepath.IsAbs(pattern) || strings.HasPrefix(normalizedPattern, "/") {
			// 规范化绝对路径，移除末尾斜杠
			normalizedAbsPathClean := strings.TrimSuffix(normalizedAbsPath, "/")
			// 检查绝对路径是否以模式开头（匹配目录及其所有内容）
			if strings.HasPrefix(normalizedAbsPathClean, normalizedPattern) {
				// 确保是精确匹配或子路径匹配（避免部分匹配）
				if normalizedAbsPathClean == normalizedPattern || strings.HasPrefix(normalizedAbsPathClean, normalizedPattern+"/") {
					return true
				}
			}
			// 使用 glob 匹配绝对路径
			matched, err := filepath.Match(pattern, absPath)
			if err == nil && matched {
				return true
			}
		}

		// 如果模式是相对路径，检查相对路径匹配
		// 规范化相对路径，移除末尾斜杠
		normalizedRelPathClean := strings.TrimSuffix(normalizedRelPath, "/")

		// 检查完整相对路径是否匹配
		matched, err := filepath.Match(pattern, relPath)
		if err == nil && matched {
			return true
		}

		// 检查相对路径是否以模式开头（用于目录匹配，如 package 或 package/）
		if strings.HasPrefix(normalizedRelPathClean, normalizedPattern) {
			// 确保是精确匹配或子路径匹配（避免部分匹配）
			if normalizedRelPathClean == normalizedPattern || strings.HasPrefix(normalizedRelPathClean, normalizedPattern+"/") {
				return true
			}
		}

		// 检查路径的各个部分是否匹配
		parts := strings.Split(normalizedRelPath, "/")
		for _, part := range parts {
			matched, err := filepath.Match(pattern, part)
			if err == nil && matched {
				return true
			}
		}

		// 支持 glob 模式匹配（如 *.log）
		matched, err = filepath.Match(pattern, filepath.Base(relPath))
		if err == nil && matched {
			return true
		}
	}

	return false
}

// CompressFile 压缩单个文件（支持 zstd、gzip 或不压缩）
// sourceFile: 源文件路径
// outputFile: 输出文件路径
// compressMethod: 压缩方式 (zstd/gzip/none)，默认为 zstd
func CompressFile(sourceFile, outputFile string, compressMethod string) error {
	// 打开源文件
	source, err := os.Open(sourceFile)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer source.Close()

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 根据压缩方式创建压缩 writer
	var compressWriter io.WriteCloser
	switch compressMethod {
	case "gzip":
		compressWriter = gzip.NewWriter(outFile)
	case "zstd", "":
		// 默认为 zstd
		zstdWriter, err := zstd.NewWriter(outFile)
		if err != nil {
			return fmt.Errorf("创建 zstd writer 失败: %v", err)
		}
		compressWriter = zstdWriter
	case "none":
		// 不压缩，直接使用文件
		compressWriter = &nopCloser{Writer: outFile}
	default:
		return fmt.Errorf("不支持的压缩方式: %s，支持的方式: zstd, gzip, none", compressMethod)
	}
	defer compressWriter.Close()

	// 复制文件内容到压缩 writer
	_, err = io.Copy(compressWriter, source)
	if err != nil {
		return fmt.Errorf("压缩文件失败: %v", err)
	}

	return nil
}

// CompressFiles 压缩多个文件到一个 tar 归档中（支持 zstd、gzip 或不压缩）
// sourceFiles: 源文件路径列表
// outputFile: 输出文件路径
// compressMethod: 压缩方式 (zstd/gzip/none)，默认为 zstd
func CompressFiles(sourceFiles []string, outputFile string, compressMethod string) error {
	if len(sourceFiles) == 0 {
		return fmt.Errorf("没有指定要压缩的文件")
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %v", err)
	}
	defer outFile.Close()

	// 根据压缩方式创建压缩 writer
	var compressWriter io.WriteCloser
	switch compressMethod {
	case "gzip":
		compressWriter = gzip.NewWriter(outFile)
	case "zstd", "":
		// 默认为 zstd
		zstdWriter, err := zstd.NewWriter(outFile)
		if err != nil {
			return fmt.Errorf("创建 zstd writer 失败: %v", err)
		}
		compressWriter = zstdWriter
	case "none":
		// 不压缩，直接使用文件
		compressWriter = &nopCloser{Writer: outFile}
	default:
		return fmt.Errorf("不支持的压缩方式: %s，支持的方式: zstd, gzip, none", compressMethod)
	}
	defer compressWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(compressWriter)
	defer tarWriter.Close()

	// 遍历每个文件并添加到 tar 归档中
	for _, sourceFile := range sourceFiles {
		// 验证文件是否存在
		info, err := os.Stat(sourceFile)
		if err != nil {
			return fmt.Errorf("文件不存在: %s, %v", sourceFile, err)
		}
		if info.IsDir() {
			return fmt.Errorf("路径是目录而不是文件: %s", sourceFile)
		}

		// 获取文件的绝对路径
		absPath, err := filepath.Abs(sourceFile)
		if err != nil {
			return fmt.Errorf("获取绝对路径失败: %v", err)
		}

		// 创建 tar header，使用文件名（不含路径）作为归档中的名称
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("创建tar header失败: %v", err)
		}
		// 使用文件名作为归档中的名称，避免路径问题
		header.Name = filepath.Base(sourceFile)

		// 写入header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("写入tar header失败: %v", err)
		}

		// 打开文件
		file, err := os.Open(absPath)
		if err != nil {
			return fmt.Errorf("打开文件失败: %v", err)
		}

		// 复制文件内容
		_, err = io.Copy(tarWriter, file)
		file.Close()
		if err != nil {
			return fmt.Errorf("复制文件内容失败: %v", err)
		}
	}

	return nil
}

// nopCloser 是一个包装器，将 io.Writer 转换为 io.WriteCloser（Close 方法为空操作）
type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

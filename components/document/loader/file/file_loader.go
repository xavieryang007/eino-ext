package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/components/document/parser"
	"code.byted.org/flow/eino/schema"
)

const (
	MetaKeyFileName  = "_file_name"
	MetaKeyExtension = "_extension"
	MetaKeySource    = "_source"
)

type FileLoaderConfig struct {
	UseNameAsID bool
	Parser      parser.Parser
}

// FileLoader loads a local file and use its content directly as Document's content.
type FileLoader struct {
	FileLoaderConfig
}

// NewFileLoader creates a new FileLoader.
func NewFileLoader(ctx context.Context, config *FileLoaderConfig) (*FileLoader, error) {
	if config == nil {
		config = &FileLoaderConfig{}
	}
	if config.Parser == nil {
		parser, err := parser.NewExtParser(ctx,
			&parser.ExtParserConfig{
				FallbackParser: parser.TextParser{},
			},
		)
		if err != nil {
			return nil, fmt.Errorf("new file parser fail: %w", err)
		}

		config.Parser = parser
	}

	return &FileLoader{FileLoaderConfig: *config}, nil
}

func (f *FileLoader) Load(ctx context.Context, src document.Source, opts ...document.LoaderOption) (docs []*schema.Document, err error) {
	defer func() {
		if err != nil {
			_ = callbacks.OnError(ctx, err)
		}
	}()

	ctx = callbacks.OnStart(ctx, &document.LoaderCallbackInput{
		Source: src,
	})

	file, err := openFile(src.URI)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	name := filepath.Base(src.URI)
	ext := filepath.Ext(src.URI)

	meta := map[string]any{
		MetaKeyExtension: ext,
		MetaKeyFileName:  name,
		MetaKeySource:    src.URI,
	}

	if f.Parser == nil {
		return nil, errors.New("no parser specified")
	}

	docs, err = f.Parser.Parse(ctx, file, parser.WithURI(src.URI), parser.WithExtraMeta(meta))
	if err != nil {
		return nil, fmt.Errorf("file parse err of [%s]: %w", src.URI, err)
	}

	if f.UseNameAsID {
		if len(docs) == 1 {
			docs[0].ID = name
		} else {
			for idx, doc := range docs {
				doc.ID = fmt.Sprintf("%s_%d", name, idx)
			}
		}
	}

	_ = callbacks.OnEnd(ctx, &document.LoaderCallbackOutput{
		Source: src,
		Docs:   docs,
	})

	return docs, nil
}

func (f *FileLoader) GetType() string {
	return "FileLoader"
}

func (f *FileLoader) IsCallbacksEnabled() bool {
	return true
}

func openFile(path string) (io.ReadCloser, error) {
	if err := validateSingleFilePath(path); err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("flat loader open file path failed with err: %w, path= %s", err, path)
	}

	return f, nil
}

func validateSingleFilePath(path string) error {
	if len(path) == 0 {
		return errors.New("read single file from path, path is empty")
	}

	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read single file from path, error while checking file stat: %w, path= %s", err, path)
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("read single file from path can only accept non-dir path, actual= %s", path)
	}

	return nil
}

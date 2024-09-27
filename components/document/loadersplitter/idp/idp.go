// Package idp implement document.LoaderSplitter in eino of IDP service.
// see more info at: https://bytedance.larkoffice.com/wiki/DstEwtq34iHkw9kEZVLc9h34n0g
package idp

import (
	"context"
	"fmt"
	"time"

	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/lark_ai/idp_sdk/config"
	"code.byted.org/lark_ai/idp_sdk/consts"
	idpSvc "code.byted.org/lark_ai/idp_sdk/service/v2"
	"code.byted.org/overpass/lark_ai_davinci/kitex_gen/lark/ai/davinci/entity"
	fentity "code.byted.org/overpass/larkai_workflow_facade/kitex_gen/larkai/workflow/facade/entity"
)

var _ document.LoaderSplitter = &LoaderSplitter{}

type idpFileType string

const (
	FileTypeOfPDF      idpFileType = "pdf"
	FileTypeOfDOCX     idpFileType = "docx"
	FileTypeOfDOC      idpFileType = "doc"
	FileTypeOfTXT      idpFileType = "txt"
	FileTypeOfMarkdown idpFileType = "markdown"
	FileTypeOfLarkDoc  idpFileType = "lark_docs"
	FileTypeOfHTML     idpFileType = "html"
	FileTypeOfPPTX     idpFileType = "pptx"
)

// Config IDP config.
type Config struct {
	// identifier of IDP.
	AppID     int64  `json:"app_id"`
	SecretKey string `json:"secret_key"`

	Strategy  fentity.ChunkingStrategy `json:"strategy"`
	ChunkSize int64                    `json:"chunk_size"`

	// file type, e.g. pdf, docx, pptx, lark_docs.
	// using IDPFileTypeXXX consts, e.g. FileTypeOfLarkDoc for lark docs.
	FileType idpFileType `json:"file_type"`

	// client timeout
	Timeout time.Duration `json:"timeout"`

	// biz uid
	BizUID string `json:"biz_uid"`

	Extra map[string]string `json:"extra"`

	// create lark doc loader if FileType is `FileTypeOfLarkDoc`
	LarkDocConfig *LarkDocConfig `json:"lark_doc_config"`
}

// LoaderSplitter IDP loader splitter.
// Load and split document using IDP service.
// refs: https://bytedance.larkoffice.com/wiki/DstEwtq34iHkw9kEZVLc9h34n0g
type LoaderSplitter struct {
	config *Config

	larkDocLoader *larkDocLoader
}

// NewIDPLoaderSplitter creates a new IDP loader splitter.
func NewIDPLoaderSplitter(config *Config) *LoaderSplitter {
	if config == nil {
		config = &Config{}
	}

	idp := &LoaderSplitter{
		config: config,
	}

	if config.LarkDocConfig != nil {
		idp.larkDocLoader = newLarkDocLoader(LarkDocConfig{
			AppID:         idp.config.LarkDocConfig.AppID,
			AppSecret:     idp.config.LarkDocConfig.AppSecret,
			BaseURL:       idp.config.LarkDocConfig.BaseURL,
			UserAccessKey: idp.config.LarkDocConfig.UserAccessKey,
		})
	}

	return idp
}

// LoadAndSplit loads and splits document.
// Document content only contains text content.
// other infos will be set in document meta data.
// e.g. block_id, positions, image_detail, etc.
func (s *LoaderSplitter) LoadAndSplit(ctx context.Context, src document.Source, opts ...document.LoaderSplitterOption) (docs []*schema.Document, err error) {
	docInfo := &fentity.DocInfoV2{
		Strategy:  s.config.Strategy,
		ChunkSize: s.config.ChunkSize,
		FileType:  string(s.config.FileType),
	}

	larkDocUserAccessKey := ""
	if s.config.LarkDocConfig != nil {
		larkDocUserAccessKey = s.config.LarkDocConfig.UserAccessKey
	}

	idpOption := &option{
		LarkDocAccessKey: larkDocUserAccessKey,
	}

	idpOption = document.GetImplSpecificOptions(idpOption, opts...)

	if s.config.FileType == FileTypeOfLarkDoc {
		if s.larkDocLoader == nil {
			return nil, fmt.Errorf("larkDoc loader is nil")
		}

		larkDocs, err := s.larkDocLoader.Load(ctx, src, idpOption)
		if err != nil {
			return nil, err
		}

		if len(larkDocs) != 1 || larkDocs[0] == nil {
			return nil, fmt.Errorf("get larkDoc failed of nil docs")
		}

		docInfo.Data = []byte(larkDocs[0].Content)
	} else {
		docInfo.URL = src.URI
	}

	conf := config.NewConfig(entity.App(s.config.AppID), s.config.SecretKey)
	docAIService := idpSvc.NewDocAIService(conf)

	taskID, err := docAIService.DocChunking(ctx, docInfo, nil, s.config.Extra)
	if err != nil {
		return nil, err
	}

	maxTimes := int(s.config.Timeout.Seconds())
	if maxTimes < 1 {
		maxTimes = 5
	}

	for i := 0; i < maxTimes; i++ {
		chunks, status, err := docAIService.QueryWorkflowTask(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunks, err: %v", err)
		}
		if status == consts.StatusRunning {
			time.Sleep(time.Second * 1)
			continue
		}

		if status == consts.StatusFailed {
			return nil, fmt.Errorf("get workflow task status failed, err: %v", err)
		}

		for _, chunk := range chunks.Chunks {
			docs = append(docs, &schema.Document{
				Content:  chunk.GetText(),
				MetaData: s.toMetaData(&chunk),
			})
		}
		break
	}

	return docs, nil
}

func (s *LoaderSplitter) toMetaData(chunk *fentity.Chunk) map[string]any {
	meta := map[string]any{}

	if chunk.ImageDetail != nil {
		meta["image_detail"] = chunk.ImageDetail
	}
	if chunk.TableDetail != nil {
		meta["table_detail"] = chunk.TableDetail
	}
	if chunk.Positions != nil {
		meta["positions"] = chunk.Positions
	}
	if chunk.Token != nil {
		meta["token"] = chunk.Token
	}
	if chunk.Children != nil {
		meta["children"] = chunk.Children
	}
	if chunk.Parent != nil {
		meta["parent"] = chunk.Parent
	}
	if chunk.Label != nil {
		meta["label"] = chunk.Label
	}
	if chunk.Level != nil {
		meta["level"] = chunk.Level
	}
	if chunk.Type != nil {
		meta["type"] = chunk.Type
	}
	if chunk.SlideIndex != nil {
		meta["slide_index"] = chunk.SlideIndex
	}
	if chunk.BlockID != nil {
		meta["block_id"] = chunk.BlockID
	}
	if chunk.ID != nil {
		meta["id"] = chunk.ID
	}
	return meta
}

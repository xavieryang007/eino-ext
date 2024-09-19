package idp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkDocx "github.com/larksuite/oapi-sdk-go/v3/service/docx/v1"

	"code.byted.org/flow/eino/components/document"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/lang/conv"
)

const typ = "LarkDoc"

type larkDocObjectType string

const (
	larkDocObjectTypeDocs     larkDocObjectType = "doc"
	larkDocObjectTypeDocx     larkDocObjectType = "docx"
	larkDocObjectTypeSheet    larkDocObjectType = "sheet"
	larkDocObjectTypeMindnote larkDocObjectType = "mindnote"
	larkDocObjectTypeBitable  larkDocObjectType = "bitable"
	larkDocObjectTypeFile     larkDocObjectType = "file"
	larkDocObjectTypeSlides   larkDocObjectType = "slides"
	larkDocObjectTypeWiki     larkDocObjectType = "wiki"
)

var larkDocObjectTypeMap = map[larkDocObjectType]bool{
	larkDocObjectTypeDocx: true,

	// TODO NOT IMPLEMENT YET
	larkDocObjectTypeDocs:     false,
	larkDocObjectTypeSheet:    false,
	larkDocObjectTypeMindnote: false,
	larkDocObjectTypeBitable:  false,
	larkDocObjectTypeFile:     false,
	larkDocObjectTypeSlides:   false,
	larkDocObjectTypeWiki:     false,
}

type LarkDocConfig struct {
	// feishu app identifier in openapi platform.
	AppID     string `json:"app_id"`
	AppSecret string `json:"app_secret"`

	// default access key, will be overwritten by `WithLarkDocAccessKey()` option
	UserAccessKey string `json:"user_access_key"`

	// base url of feishu openapi, default is https://open.feishu.cn
	// refs: https://github.com/larksuite/oapi-sdk-go/blob/v3_main/README.md#%E9%85%8D%E7%BD%AEapi-client
	BaseURL string `json:"base_url"`
}

// LarkDocLoader.
// load document from lark doc.
// return document is lark blocks marshalled as json.
type larkDocLoader struct {
	config *LarkDocConfig

	cli *lark.Client
}

func newLarkDocLoader(config LarkDocConfig) *larkDocLoader {
	cli := newLarkClient(config.AppID, config.AppSecret, config.BaseURL)

	l := &larkDocLoader{
		config: &config,
		cli:    cli,
	}

	return l
}

func (l *larkDocLoader) Load(ctx context.Context, src document.Source, opt *option) ([]*schema.Document, error) {
	if src.URI == "" {
		return nil, errors.New("lark doc uri is empty")
	}

	docToken, docType, err := parseLarkURL(src.URI)
	if err != nil {
		return nil, err
	}

	var docs []*schema.Document

	switch docType {
	case larkDocObjectTypeDocx:

		pageToken := ""
		hasMore := true

		docBlocks := make([]*larkDocx.Block, 0)

		for hasMore {
			reqBuilder := larkDocx.NewListDocumentBlockReqBuilder().DocumentId(docToken)
			if pageToken != "" {
				reqBuilder.PageToken(pageToken)
			}

			req := reqBuilder.Build()

			blockRes, err := l.cli.Docx.V1.DocumentBlock.List(ctx, req, larkcore.WithUserAccessToken(opt.LarkDocAccessKey))
			if err != nil {
				return nil, err
			}
			if !blockRes.Success() {
				return nil, fmt.Errorf("failed to list doc blocks: %s", blockRes.Msg)
			}

			docBlocks = append(docBlocks, blockRes.Data.Items...)

			pageToken = conv.StringDefault(blockRes.Data.PageToken, "")
			hasMore = conv.BoolDefault(blockRes.Data.HasMore, false)
		}

		v, err := json.Marshal(docBlocks)
		if err != nil {
			return nil, err
		}

		docs = append(docs, &schema.Document{
			Content: string(v),
			ID:      docToken,
			MetaData: map[string]any{
				"type": larkDocObjectTypeDocx,
			},
		})

	default:
		return nil, errors.New("unsupported lark doc type")
	}

	return docs, nil
}

func (l *larkDocLoader) IsCallbacksEnabled() bool {
	return true
}

func (l *larkDocLoader) GetType() string {
	return typ
}

var accessTokenGrantType = "authorization_code"

func newLarkClient(appID, appSecret, baseURL string) *lark.Client {
	opts := make([]lark.ClientOptionFunc, 0, 1)
	if len(baseURL) > 0 {
		opts = append(opts, lark.WithOpenBaseUrl(baseURL))
	}

	client := lark.NewClient(appID, appSecret,
		opts...,
	)

	return client
}

// parseLarkURL parse lark doc url.
// returns docToken, docType, error.
func parseLarkURL(urlStr string) (string, larkDocObjectType, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", err
	}
	path := u.Path
	splits := strings.Split(path, "/")
	splits = removeEmptyString(splits)

	length := len(splits)

	if length < 2 {
		return "", "", errors.New("invalid lark url")
	}

	docType := larkDocObjectType(splits[length-2])

	isSupport := larkDocObjectTypeMap[docType]
	if !isSupport {
		return "", "", errors.New("unsupported lark doc type")
	}

	return splits[length-1], docType, nil
}

func removeEmptyString(slice []string) []string {
	newSlice := make([]string, 0, len(slice))
	for _, elem := range slice {
		if len(elem) > 0 {
			newSlice = append(newSlice, elem)
		}
	}

	return newSlice
}

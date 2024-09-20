package fornaxplugin

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/asynccache"
	"code.byted.org/gopkg/logs/v2"
)

var asyncCache asynccache.AsyncCache

func init() {
	opt := asynccache.Options{
		RefreshDuration: 5 * time.Second,
		Fetcher:         fetcher,

		EnableExpire:   true,
		ExpireDuration: 2 * 24 * time.Hour,
		DeleteHandler:  deleteHandler,

		ErrorHandler: errorHandler,
		ErrLogFunc:   errLogFunc,
	}
	asyncCache = asynccache.NewAsyncCache(opt)
}

type keyInfo struct {
	toolID int64
}

func getToolInfo(_ context.Context, toolID int64) (pi *schema.ToolInfo, err error) {

	ki := keyInfo{
		toolID: toolID,
	}

	key := encode(ki)
	val, err := asyncCache.Get(key)
	if err != nil {
		// when error occurs during the first time fetching, it should fetch again in next request rather than throw an error cached
		asyncCache.DeleteIf(func(k string) bool {
			return key == k
		})
		return nil, fmt.Errorf("[Fornax Plugin]get tool from asynccache failed, toolID=%v, err=%v", key, err)
	}

	toolInfo, ok := val.(*schema.ToolInfo)
	if !ok {
		return nil, fmt.Errorf("[Fornax Plugin]expected type of *schema.ToolInfo, got %T", val)
	}

	return toolInfo, nil
}

func fetcher(key string) (interface{}, error) {
	ki, err := decode(key)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	toolsInfo, err := getToolDescription(ctx, []int64{ki.toolID})
	if err != nil {
		return nil, err
	}

	toolDef, ok := toolsInfo[ki.toolID]
	if !ok || toolDef == nil {
		return nil, fmt.Errorf("[Fornax Plugin]tool not found, toolID=%v", ki.toolID)
	}

	toolInfo, err := convertToolInfo(ctx, toolDef)
	if err != nil {
		return nil, err
	}

	return toolInfo, nil
}

func errorHandler(key string, err error) {
	logs.Errorf("[Fornax Plugin]async cache failed, key=%v, error=%v", key, err)
}

func errLogFunc(str string) {
	logs.Errorf("[Fornax Plugin]async cache meet error: %v", str)
}

func deleteHandler(key string, oldData interface{}) {
	logs.Infof("[Fornax Plugin]deleted due to expired, key=%v", key)
}

func encode(ki keyInfo) (key string) {
	return strconv.FormatInt(ki.toolID, 10)
}

func decode(key string) (ki keyInfo, err error) {
	toolID, err := strconv.ParseInt(key, 10, 64)
	if err != nil {
		return keyInfo{}, err
	}
	return keyInfo{
		toolID: toolID,
	}, nil
}

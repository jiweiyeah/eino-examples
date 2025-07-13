/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package einoagent

import (
	"context"
	"github.com/cloudwego/eino-examples/internal/logs"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"
)

func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	// TODO Modify component configuration here.
	if cwd, err := os.Getwd(); err == nil {
		log.Println("当前工作目录:", cwd)
	} else {
		log.Println("获取当前目录失败:", err)
	}
	ARK_EMBEDDING_MODEL := os.Getenv("ARK_EMBEDDING_MODEL")
	ARK_API_KEY := os.Getenv("ARK_API_KEY")
	logs.Infof("ARK_EMBEDDING_MODEL: %s", ARK_EMBEDDING_MODEL)
	logs.Infof("ARK_API_KEY: %s", ARK_API_KEY)
	config := &ark.EmbeddingConfig{
		Model:  ARK_EMBEDDING_MODEL,
		APIKey: ARK_API_KEY,
	}

	eb, err = ark.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}

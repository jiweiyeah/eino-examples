package main

import (
	"context"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"log"
)

func buildAgent(ctx context.Context) (compile compose.Runnable[map[string]any, *schema.Message], err error) {
	//1.新建图
	newGraph := compose.NewGraph[map[string]any, *schema.Message]()
	//2.chattemplate
	chatTemplate, err := newChatTemplate(ctx)

	lba, err := newLambda(ctx) //Reactagent

	newGraph.AddChatTemplateNode("chatTemplate", chatTemplate, compose.WithNodeName("chatTemplate"))
	newGraph.AddLambdaNode("lba", lba, compose.WithNodeName("ReactAgent"))

	newGraph.AddEdge(compose.START, "chatTemplate")
	newGraph.AddEdge("chatTemplate", "lba")
	newGraph.AddEdge("lba", compose.END)

	compile, err = newGraph.Compile(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return compile, nil
}

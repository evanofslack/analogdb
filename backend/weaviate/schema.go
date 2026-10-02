package weaviate

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) CreateSchemas(ctx context.Context) error {
	err := ss.db.createSchemas(ctx)
	return err
}

func (db *DB) createSchemas(ctx context.Context) error {
	err := db.createPostImageSchema(ctx)
	return err
}

func (db *DB) createPostImageSchema(ctx context.Context) error {
	db.logger.DebugContext(ctx, "Start create post image schema in vector db")

	exists, err := db.db.Schema().ClassExistenceChecker().WithClassName(PostImageClass).Do(ctx)
	if err != nil {
		err = fmt.Errorf("check post image schema, %w", err)
		db.logger.ErrorContext(ctx, "Fail check post image schema in vector db", "error", err)
		return err
	}
	if exists {
		db.logger.DebugContext(ctx, "Found post image schema in vector db")
		return nil
	}

	err = db.db.Schema().ClassCreator().WithClass(postImageClass()).Do(ctx)
	if err != nil {
		err = fmt.Errorf("create post image schema, %w", err)
		db.logger.ErrorContext(ctx, "Fail create post image schema in vector db", "error", err)
		return err
	}

	db.logger.InfoContext(ctx, "Created post image schema in vector db")
	return nil
}

func postImageClass() *models.Class {
	searchable := true
	return &models.Class{
		Class:       PostImageClass,
		Description: "Analog photographs",
		ModuleConfig: map[string]any{
			"multi2vec-clip": map[string]any{
				"imageFields": []string{"image"},
			},
		},
		VectorIndexType: "hnsw",
		Vectorizer:      "multi2vec-clip",
		VectorIndexConfig: map[string]any{
			"distance":       "cosine",
			"ef":             float64(128),
			"efConstruction": float64(128),
			"maxConnections": float64(32),
		},
		Properties: []*models.Property{
			{
				Name:        "image",
				DataType:    []string{"blob"},
				Description: "image",
			},
			{
				Name:        "post_id",
				DataType:    []string{"int"},
				Description: "unique post_id",
			},
			{
				Name:            "title",
				DataType:        []string{"text"},
				Description:     "post title",
				IndexSearchable: &searchable,
				Tokenization:    "word",
			},
			{
				Name:            "caption",
				DataType:        []string{"text"},
				Description:     "post caption",
				IndexSearchable: &searchable,
				Tokenization:    "word",
			},
			{
				Name:            "tags",
				DataType:        []string{"text[]"},
				Description:     "post tags",
				IndexSearchable: &searchable,
				Tokenization:    "word",
			},
			{
				Name:        "grayscale",
				DataType:    []string{"boolean"},
				Description: "is post grayscale",
			},
			{
				Name:        "nsfw",
				DataType:    []string{"boolean"},
				Description: "is post nsfw",
			},
			{
				Name:        "sprocket",
				DataType:    []string{"boolean"},
				Description: "is post sprocket",
			},
		},
	}
}

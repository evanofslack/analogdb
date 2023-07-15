package weaviate

import (
	"context"
	"fmt"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/schema"
	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) CreateSchemas(ctx context.Context) error {
	err := ss.db.createSchemas(ctx)
	return err
}

func (db *DB) createSchemas(ctx context.Context) error {
	err := db.createPictureSchema(ctx)
	return err
}

func (db *DB) getSchema(ctx context.Context) (*schema.Dump, error) {
	schema, err := db.db.Schema().Getter().Do(ctx)
	if err != nil {
		return nil, err
	}
	return schema, nil
}

func (db *DB) createPictureSchema(ctx context.Context) error {

	db.logger.Debug().Msg("Starting to create picture schema in vector DB")

	classObj := &models.Class{
		Class:       "Picture",
		Description: "Analog photographs",
		ModuleConfig: map[string]any{
			"img2vec-neural": map[string]any{
				"imageFields": []string{"image"},
			},
		},
		VectorIndexType: "hnsw",
		Vectorizer:      "img2vec-neural",
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

	err := db.db.Schema().ClassCreator().WithClass(classObj).Do(context.Background())
	if err != nil {
		err = fmt.Errorf("Failed to create picture schema, %w", err)
		db.logger.Error().Err(err).Msg("Failed to create picture schema in vector DB")
		return err
	}

	db.logger.Info().Msg("Created picture schema in vector DB")
	return nil
}

package weaviate

import (
	"context"

	"github.com/weaviate/weaviate/entities/models"
)

func (ss SimilarityService) CreateSchemas(ctx context.Context) error {
	err := ss.db.createPictureSchema(ctx)
	return err
}

func (db *DB) createPictureSchema(ctx context.Context) error {

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
				Name:        "id",
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
	return err
}

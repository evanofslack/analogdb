import dagster as dg

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    colors,
    final_posts,
    keywords,
    patch_posts_scores,
    reddit_posts,
    s3_images,
    title_metadatas,
    updated_posts_scores,
    upload_posts,
)

scrape_job = dg.define_asset_job(
    name="scrape_and_upload",
    selection=[
        analogdb_permalinks,
        reddit_posts,
        title_metadatas,
        s3_images,
        colors,
        keywords,
        final_posts,
        upload_posts,
    ],
)

patch_scores_job = dg.define_asset_job(
    name="update_post_scores",
    selection=[
        analogdb_posts,
        updated_posts_scores,
        patch_posts_scores,
    ],
)

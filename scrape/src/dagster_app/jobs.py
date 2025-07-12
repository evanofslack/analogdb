import dagster as dg

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    colors,
    final_posts,
    keywords,
    patch_post_descriptions,
    patch_post_keywords,
    patch_post_scores,
    patch_post_title_metadatas,
    reddit_comments_to_s3,
    reddit_posts,
    s3_images,
    title_metadatas,
    updated_post_descriptions,
    updated_post_keywords,
    updated_post_scores,
    updated_post_title_metadatas,
    updated_reddit_comments,
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
        updated_post_scores,
        patch_post_scores,
    ],
)

patch_descriptions_job = dg.define_asset_job(
    name="update_post_descriptions",
    selection=[
        analogdb_posts,
        updated_post_descriptions,
        patch_post_descriptions,
    ],
)

patch_keywords_job = dg.define_asset_job(
    name="update_post_keywords",
    selection=[
        analogdb_posts,
        updated_reddit_comments,
        reddit_comments_to_s3,
        updated_post_keywords,
        patch_post_keywords,
    ],
)

patch_post_title_metadatas_job = dg.define_asset_job(
    name="update_post_title_metadatas_descriptions",
    selection=[
        analogdb_posts,
        updated_post_title_metadatas,
        patch_post_title_metadatas,
    ],
)

import dagster as dg

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    archive_final_posts,
    archive_post_images,
    archive_post_tags,
    archive_reddit_posts,
    archive_title_metadatas,
    archive_upload_posts,
    final_posts,
    patch_post_descriptions,
    patch_post_metadata,
    patch_post_scores,
    post_images,
    post_tags,
    reddit_comments_to_s3,
    reddit_posts,
    title_metadatas,
    updated_post_descriptions,
    updated_post_scores,
    updated_reddit_comments,
    upload_posts,
)
from .backfill import (
    reencode_post_vectors,
    reextract_post_colors,
    rematch_post_metadata,
    tag_missing_post_captions,
)
from .duplicates import report_duplicates

scrape_job = dg.define_asset_job(
    name="scrape_and_upload",
    selection=[
        analogdb_permalinks,
        reddit_posts,
        title_metadatas,
        post_images,
        post_tags,
        final_posts,
        upload_posts,
    ],
)

archive_job = dg.define_asset_job(
    name="scrape_archive",
    description="Fill missed days from Arctic Shift, set config.days to dates or ranges",
    tags={"reddit": "true"},
    selection=[
        archive_reddit_posts,
        archive_title_metadatas,
        archive_post_images,
        archive_post_tags,
        archive_final_posts,
        archive_upload_posts,
    ],
)

patch_scores_job = dg.define_asset_job(
    name="update_post_scores",
    tags={"reddit": "true"},
    selection=[
        analogdb_posts,
        updated_post_scores,
        patch_post_scores,
    ],
)

patch_descriptions_job = dg.define_asset_job(
    name="update_post_descriptions",
    tags={"reddit": "true"},
    selection=[
        analogdb_posts,
        updated_post_descriptions,
        patch_post_descriptions,
    ],
)

patch_comments_job = dg.define_asset_job(
    name="update_post_comments",
    tags={"reddit": "true"},
    selection=[
        analogdb_posts,
        updated_reddit_comments,
        reddit_comments_to_s3,
        patch_post_metadata,
    ],
)


rematch_metadata_job = dg.define_asset_job(
    name="rematch_metadata",
    selection=[rematch_post_metadata],
)

tag_missing_captions_job = dg.define_asset_job(
    name="tag_missing_captions",
    selection=[tag_missing_post_captions],
)

reencode_vectors_job = dg.define_asset_job(
    name="reencode_vectors",
    selection=[reencode_post_vectors],
)

reextract_colors_job = dg.define_asset_job(
    name="reextract_colors",
    selection=[reextract_post_colors],
)

report_duplicates_job = dg.define_asset_job(
    name="report_duplicate_posts",
    selection=[report_duplicates],
)

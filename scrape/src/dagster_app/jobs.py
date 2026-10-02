import dagster as dg

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
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
    backfill_post_captions,
    backfill_post_metadata,
    reencode_post_vectors,
    rematch_post_metadata,
)

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


backfill_metadata_job = dg.define_asset_job(
    name="backfill_metadata",
    tags={"reddit": "true"},
    selection=[backfill_post_metadata],
)

rematch_metadata_job = dg.define_asset_job(
    name="rematch_metadata",
    selection=[rematch_post_metadata],
)

backfill_captions_job = dg.define_asset_job(
    name="backfill_captions",
    selection=[backfill_post_captions],
)

reencode_vectors_job = dg.define_asset_job(
    name="reencode_vectors",
    selection=[reencode_post_vectors],
)

UPDATE pictures SET
    url = CASE WHEN url ~ '^https?://' THEN url ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || url END,
    lowurl = CASE WHEN lowurl ~ '^https?://' OR lowurl = '' THEN lowurl ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || lowurl END,
    medurl = CASE WHEN medurl ~ '^https?://' OR medurl = '' THEN medurl ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || medurl END,
    highurl = CASE WHEN highurl ~ '^https?://' OR highurl = '' THEN highurl ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || highurl END;

UPDATE removed_posts SET
    url = CASE WHEN url ~ '^https?://' THEN url ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || url END,
    low_url = CASE WHEN low_url ~ '^https?://' OR low_url = '' THEN low_url ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || low_url END,
    med_url = CASE WHEN med_url ~ '^https?://' OR med_url = '' THEN med_url ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || med_url END,
    high_url = CASE WHEN high_url ~ '^https?://' OR high_url = '' THEN high_url ELSE 'https://d3i73ktnzbi69i.cloudfront.net/' || high_url END;

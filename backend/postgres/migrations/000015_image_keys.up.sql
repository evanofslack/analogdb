-- image columns hold the S3 key, the API builds URLs from its image base
UPDATE pictures SET
    url = regexp_replace(url, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    lowurl = regexp_replace(lowurl, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    medurl = regexp_replace(medurl, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    highurl = regexp_replace(highurl, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', '');

UPDATE removed_posts SET
    url = regexp_replace(url, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    low_url = regexp_replace(low_url, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    med_url = regexp_replace(med_url, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', ''),
    high_url = regexp_replace(high_url, '^https://(i\.analogdb\.com|d3i73ktnzbi69i\.cloudfront\.net)/', '');

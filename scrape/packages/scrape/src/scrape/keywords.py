import spacy
import string
import math
from pathlib import Path
from collections import Counter
from typing import List, Optional, Set
from functools import cached_property
from .models import RedditComment, Keyword


class KeywordBlacklist:
    def __init__(self, file_path: str):
        self.file_path = file_path

    @cached_property
    def blacklist(self) -> Set[str] | None:
        file_path = Path(self.file_path)
        if not file_path.exists():
            return None
        with open(file_path, "r", encoding="utf-8") as f:
            words = [line.strip() for line in f.readlines()]
            return set(word for word in words if word)


class KeywordExtractor:
    """Analyzes reddit title and comments to extract and rank keywords based"""

    def __init__(self):
        self._nlp = spacy.load("en_core_web_lg")
        self._union_blacklist = {
            "http",
            "www",
            ".com",
            "imgur",
            "wikapedia",
            "u/",
            "r/",
        }
        self._punctuation = r"""!"#$%&'()*,/:;<=>?@[\]^_`{|}~"""
        self._pos_tags = ["PROPN", "ADJ", "NOUN"]
        self._printable = set(string.printable)

    def post_keywords(
        self,
        title: str,
        score: int,
        comments: List[RedditComment],
        limit: Optional[int] = None,
        blacklist: Optional[Set[str]] = None,
    ) -> List[Keyword]:
        """
        Extract and rank keywords from a Reddit post (title + comments).
        """
        combined = self._title_counter(
            title=title, score=score
        ) + self._comment_counter(comments=comments)

        if blacklist is not None:
            combined = self._remove_from_counter(counter=combined, blacklist=blacklist)

        keywords = self._counter_to_keywords(counter=combined, limit=limit)
        return keywords

    def _extract_keywords(
        self, text: str, blacklist: Optional[Set[str]] = None
    ) -> List[str]:
        """
        Extract keywords from text using NLP processing.
        """
        keywords = []
        doc = self._nlp(text.lower())

        for token in doc:
            # Skip stop words
            if token.text in self._nlp.Defaults.stop_words:
                continue
            # Skip single characters
            if len(token.text) < 2:
                continue
            # Skip punctuation
            if bool(set(token.text) & set(self._punctuation)):
                continue
            # Skip non-printable characters
            if not set(token.text).issubset(self._printable):
                continue
            # Skip blacklisted words
            if blacklist and token.text in blacklist:
                continue
            # Skip union blacklist
            if token.text in self._union_blacklist:
                continue
            # Only keep specified POS tags
            if token.pos_ not in self._pos_tags:
                continue

            keywords.append(token.text)

        return keywords

    def _rank_keywords(self, keywords: List[str], weight: int) -> Counter:
        """Apply weight multiplier to keyword frequency counts."""
        count = Counter(keywords)

        # No weight, no need to iterate
        if weight <= 1:
            return count

        # Multiply count by weight
        for word, _ in count.most_common():
            count[word] *= weight

        return count

    def _counter_to_keywords(
        self, counter: Counter, limit: Optional[int] = None
    ) -> List[Keyword]:
        """Convert Counter to AnalogKeyword objects with normalized weights."""
        keywords: List[Keyword] = []

        total = counter.total()
        for word, score in counter.most_common(n=limit):
            keyword = Keyword(word=word, weight=score / total)
            keywords.append(keyword)

        return keywords

    def _remove_from_counter(self, counter: Counter, blacklist: Set[str]) -> Counter:
        """Remove blacklisted words from Counter."""
        remove = [word for word in counter.keys() if word in blacklist]
        for word in remove:
            del counter[word]
        return counter

    def _comment_counter(self, comments: List[RedditComment]) -> Counter:
        """Process comments to extract and weight keywords based on comment scores."""
        ranked_keywords: Counter = Counter()

        for comment in comments:
            keywords = self._extract_keywords(comment.body)

            # Avoid invalid logarithms
            if int(comment.score) <= 1:
                weight = 1
            else:
                weight = int(math.log(int(comment.score), 2) * 100)

            ranked = self._rank_keywords(keywords=keywords, weight=weight)
            ranked_keywords += ranked

        return ranked_keywords

    def _title_counter(self, title: str, score: int) -> Counter:
        """Process post title to extract and weight keywords based on post score."""
        keywords = self._extract_keywords(title)

        # Avoid invalid logarithms
        if score <= 1:
            weight = 1
        else:
            weight = int(math.log(int(score), 10) * 100)

        ranked_title = self._rank_keywords(keywords=keywords, weight=weight)
        return ranked_title

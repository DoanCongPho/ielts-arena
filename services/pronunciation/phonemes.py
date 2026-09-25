"""Reference pronunciations: what each word is expected to sound like.

Every word gets several references — British and American English, each in
its connected-speech form (phonemized in context: "a" as /ə/) and its
citation form (on its own: "a" as /eɪ/), plus the common weak forms of
function words — and is scored against whichever it matches best, so
neither accent nor a careful strong form costs anything.
"""

from __future__ import annotations

from dataclasses import dataclass

from phonemizer.backend import EspeakBackend
from phonemizer.separator import Separator

import scoring
from model import UNK, tokenizer

VARIANTS = ("en-us", "en-gb")

# Phones and words need different separators; words are merged back into
# one phone list anyway (a number like "2019" can expand to several words).
SEP = Separator(phone=" ", word="_", syllable="")
backends = {v: EspeakBackend(v, with_stress=True, language_switch="remove-flags") for v in VARIANTS}


@dataclass
class Reference:
    phones: list[str]
    ids: list[int]
    stressed: int  # index into the word's vowels of the primary stress, -1 if unknown


def to_ids(phones: list[str]) -> tuple[list[str], list[int]]:
    """Maps espeak phonemes to model tokens, splitting any the model's
    vocabulary lacks into characters and dropping what's still unknown."""
    kept, ids = [], []
    for p in phones:
        i = tokenizer.convert_tokens_to_ids(p)
        if i != UNK:
            kept.append(p)
            ids.append(i)
            continue
        for ch in p:
            j = tokenizer.convert_tokens_to_ids(ch)
            if j != UNK:
                kept.append(ch)
                ids.append(j)
    return kept, ids


def phonemize_in_context(words: list[str], variant: str) -> list[str]:
    """One phone string per word, phonemized as a whole utterance so
    function words get their weak forms ("a" as /ə/, not the letter name
    /eɪ/). Falls back to word by word when the utterance doesn't split
    back into the same number of words (a number or symbol read as
    several words)."""
    joined = backends[variant].phonemize([" ".join(words)], separator=SEP, strip=True)[0]
    per_word = [w.strip() for w in joined.split("_")]
    per_word = [w for w in per_word if w]
    if len(per_word) == len(words):
        return per_word
    return backends[variant].phonemize(words, separator=SEP, strip=True)


# Weak forms of English function words. In connected speech these are the
# normal pronunciations ("a" is /ə/, "to" is /tə/), and IELTS descriptors
# reward using them, so they must never count as mispronunciations. espeak
# can't be relied on for them: phonemizing a sentence merges words ("I am"
# becomes one), so its per-word output falls back to citation forms.
WEAK_FORMS = {
    "a": ["ə", "ɐ"], "an": ["ə n", "ɐ n"], "the": ["ð ə", "ð ɪ", "ð i"],
    "to": ["t ə", "t ʊ"], "of": ["ə v", "ə"], "and": ["ə n d", "ə n", "n"],
    "for": ["f ɚ", "f ə"], "at": ["ə t"], "as": ["ə z"], "was": ["w ə z"],
    "were": ["w ɚ", "w ə"], "can": ["k ə n", "k n"], "from": ["f ɹ ə m"],
    "that": ["ð ə t"], "some": ["s ə m"], "are": ["ɚ", "ə"], "you": ["j ə"],
    "your": ["j ɚ", "j ə"], "but": ["b ə t"], "them": ["ð ə m", "ə m"],
    "have": ["h ə v", "ə v"], "has": ["h ə z", "ə z"], "had": ["h ə d", "ə d"],
    "would": ["w ə d", "ə d"], "do": ["d ə"], "does": ["d ə z"], "am": ["ə m"],
    "or": ["ɚ"], "than": ["ð ə n"], "there": ["ð ɚ"], "his": ["ɪ z"], "her": ["h ɚ", "ɚ"],
}


def weak_references(word: str) -> list[Reference]:
    out = []
    for form in WEAK_FORMS.get(word.lower().strip(".,!?;:\"'"), []):
        phones, ids = to_ids(form.split())
        if ids:
            out.append(Reference(phones, ids, -1))
    return out


def references(words: list[str], variant: str, in_context: bool) -> list[Reference]:
    out = []
    texts = phonemize_in_context(words, variant) if in_context else backends[variant].phonemize(words, separator=SEP, strip=True)
    for text in texts:
        phones, stressed_token = [], -1
        for tok in text.replace("_", " ").split():
            if "ˈ" in tok and stressed_token < 0:
                stressed_token = len(phones)
            tok = tok.replace("ˈ", "").replace("ˌ", "")
            if tok:
                phones.append(tok)
        phones, ids = to_ids(phones)
        # The stress mark sits at the stressed syllable's start; its
        # nucleus is the first vowel from there.
        stressed = -1
        if stressed_token >= 0:
            vowel_idx = [k for k, p in enumerate(phones) if scoring.is_vowel(p)]
            after = [n for n, k in enumerate(vowel_idx) if k >= stressed_token]
            stressed = after[0] if after else -1
        out.append(Reference(phones, ids, stressed))
    return out


def all_references(texts: list[str]) -> dict[tuple[str, bool], list[Reference]]:
    """Every word's references per (variant, in_context), by word index."""
    return {(v, ctx): references(texts, v, ctx) for v in VARIANTS for ctx in (True, False)}

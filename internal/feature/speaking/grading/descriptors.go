package grading

// The four IELTS Speaking criteria, weighted equally in the overall band.
const (
	CriterionFC  = "Fluency and Coherence"
	CriterionLR  = "Lexical Resource"
	CriterionGRA = "Grammatical Range and Accuracy"
	CriterionP   = "Pronunciation"
)

var criteria = []string{CriterionFC, CriterionLR, CriterionGRA, CriterionP}

// descriptors paraphrases the public IELTS Speaking band
// descriptors (the version revised in May 2023), band 9 down to band 1.
// They are paraphrased, not quoted, but each band keeps the features an
// examiner checks for it, and keeps the descriptor's structure: Pronunciation
// bands 7, 5 and 3 are defined only relative to the bands either side.
var descriptors = map[string]string{
	CriterionFC: `Band 9: Speaks fluently; repetition or self-correction is rare. Any hesitation is for thinking about ideas, never for finding words or grammar. Cohesive devices are used fully appropriately. Topics are developed fully, coherently and at appropriate length.
Band 8: Speaks fluently with only occasional repetition or self-correction. Hesitation is usually about content and only rarely a search for language. Topics are developed coherently, appropriately and relevantly.
Band 7: Keeps going and produces long turns readily, without noticeable effort. Some hesitation, repetition or self-correction, sometimes mid-sentence, shows occasional difficulty reaching the right language, but coherence is not affected. Uses spoken discourse markers, connectives and cohesive features flexibly.
Band 6: Keeps going and is willing to produce long turns. Coherence is sometimes lost because of hesitation, repetition or self-correction. Uses a range of discourse markers, connectives and cohesive features, not always appropriately.
Band 5: Usually keeps going, but depends on repetition, self-correction or slow speech to do so. Hesitation is often a mid-sentence search for fairly basic words and grammar. Some discourse markers and connectives are overused. Simple language can be fluent, but more complex speech usually causes disfluency.
Band 4: Cannot keep going without noticeable pauses. Speech may be slow, with frequent repetition and self-correction. Links simple sentences, often repeating the same connectives. Coherence breaks down in places.
Band 3: Frequent and sometimes long pauses while searching for words. Can link simple sentences only to a limited degree and rarely goes beyond simple responses. Often cannot convey the basic message.
Band 2: Long pauses before nearly every word. A few isolated words may be recognisable, but the speech communicates almost nothing.
Band 1: No meaningful communication; speech is completely incoherent.`,

	CriterionLR: `Band 9: Uses vocabulary with total flexibility and precision on every topic. Idiomatic language is used naturally and accurately throughout.
Band 8: A wide vocabulary is used readily and flexibly to express precise meaning on every topic. Less common and idiomatic items are used skilfully, with only occasional inaccuracy in word choice or collocation. Paraphrases effectively whenever needed.
Band 7: Vocabulary is used flexibly across a variety of topics. Uses some less common and idiomatic items and shows awareness of style and collocation, though some choices are inappropriate. Paraphrases effectively.
Band 6: Enough vocabulary to discuss topics at length and make meaning clear, even when some words are used inappropriately. Paraphrases successfully in general.
Band 5: Enough vocabulary for both familiar and unfamiliar topics, but used with little flexibility. Attempts paraphrase with mixed success.
Band 4: Enough vocabulary for familiar topics, but only basic meaning on unfamiliar ones. Frequent errors and inappropriate word choices. Rarely attempts paraphrase.
Band 3: Simple vocabulary, used mainly for personal information. Not enough vocabulary for less familiar topics.
Band 2: Very limited vocabulary: isolated words or memorised phrases.
Band 1: No resource beyond a few isolated words.`,

	CriterionGRA: `Band 9: Structures are precise and accurate at all times, apart from the kind of slips native speakers also make.
Band 8: A wide range of structures, used flexibly. Most sentences are error-free. Errors are occasional, unsystematic or small inappropriacies, and a few basic errors may remain.
Band 7: A range of structures used flexibly. Error-free sentences are frequent. Simple and complex sentences are both used effectively despite some errors, and a few basic errors remain.
Band 6: Mixes short and complex sentence forms and uses a variety of structures, with limited flexibility. Complex structures often contain errors, but these rarely get in the way of communication.
Band 5: Basic sentence forms are fairly accurate. Complex structures are attempted but limited in range, nearly always contain errors, and may need to be rephrased.
Band 4: Produces basic sentence forms, and some short utterances are error-free. Subordinate clauses are rare. Turns are short, structures repetitive and errors frequent.
Band 3: Attempts basic sentence forms, but errors are numerous except in what seem to be memorised phrases.
Band 2: No evidence of basic sentence forms.
Band 1: No rateable language.`,

	CriterionP: `Band 9: Uses the full range of pronunciation features to express precise and subtle meaning. Connected speech is flexible and sustained throughout. Understood effortlessly throughout; accent has no effect on intelligibility.
Band 8: Uses a wide range of pronunciation features to express precise and subtle meaning. Keeps an appropriate rhythm. Uses stress and intonation flexibly over long utterances, with occasional lapses. Easily understood throughout; accent has minimal effect on intelligibility.
Band 7: Has every positive feature of band 6 and some, but not all, of the positive features of band 8.
Band 6: Uses a range of pronunciation features with uneven control. Chunking is generally appropriate, but rhythm may suffer from a lack of stress-timing or from speaking too fast. Some effective use of intonation and stress, not sustained. Some words or sounds are mispronounced, causing only occasional unclarity. Can generally be understood throughout without much effort.
Band 5: Has every positive feature of band 4 and some, but not all, of the positive features of band 6.
Band 4: Uses some acceptable pronunciation features, but the range is limited. Some acceptable chunking, with frequent lapses in overall rhythm. Attempts intonation and stress with limited control. Words and sounds are often mispronounced, causing unclarity. The listener needs some effort to understand, and there may be stretches that cannot be understood.
Band 3: Has some, but not all, of the positive features of band 4.
Band 2: Few acceptable pronunciation features, perhaps because the sample is too small. Delivery problems undermine connected speech. Most words and sounds are mispronounced and little meaning comes across. Often unintelligible.
Band 1: Speech cannot be understood.`,
}

// examinerPrinciples is the part of examiner practice that applies to
// every criterion.
const examinerPrinciples = `How an IELTS examiner rates:
- Rate the performance across the whole test, not answer by answer. Use every part you are given.
- Use best fit: award the band whose descriptor as a whole fits the performance better than the bands either side. A band's key features must be present to award it.
- Award whole bands only (1-9).
- Never penalise accent itself, or the opinions and ideas expressed. Rate language only.
- Rehearsed or memorised answers, and answers that don't address the question, are not evidence of ability: leave them out when rating and report their question ids.
- The transcript comes from speech recognition. It may drop fillers or quietly fix small slips, so don't reward accuracy the measurements contradict, and don't penalise a word that is clearly a recognition error.`

package ielts_test

// Criterion names are the keys of GradingResult.Criteria. The first writing
// criterion differs by task: Task 1 asks for a report on given information
// (Task Achievement), Task 2 for an argued position (Task Response).
const (
	criterionTaskAchievement = "Task Achievement"
	criterionTaskResponse    = "Task Response"
	criterionCoherence       = "Coherence and Cohesion"
	criterionLexical         = "Lexical Resource"
	criterionGrammar         = "Grammatical Range and Accuracy"
)

// minWords is the length the IELTS task asks for. An answer under it is
// penalised under Task Achievement / Task Response.
var minWords = map[string]int{"task1": 150, "task2": 250}

// correctionIssues are the categories a correction's `issue` must be one of;
// the review UI colours marks by them.
var correctionIssues = []string{"grammar", "vocabulary", "spelling", "punctuation", "cohesion", "task"}

// writingCriteria lists the four criteria a writing grade must score, in
// the order the examiner prompt presents them.
func writingCriteria(taskType string) []string {
	first := criterionTaskResponse
	if taskType == "task1" {
		first = criterionTaskAchievement
	}
	return []string{first, criterionCoherence, criterionLexical, criterionGrammar}
}

// examinerPrinciples are the rules a Writing examiner applies across every
// criterion, before any one criterion's descriptors.
const examinerPrinciples = `How IELTS Writing examiners rate:
- Each criterion gets a whole band from 0 to 9. The band is the best fit: a band is awarded only when the response shows all of that band's key features; when it shows some features of the band above but not all, it stays at the lower band.
- Rate what is on the page. Don't reward length beyond the task's requirement, memorised or formulaic language, or rare words and complex structures that are used wrongly.
- Bands 0-2: 0 for no answer, a wholly memorised response or one totally unrelated to the task; 1 for a response of under 20 words or one that barely relates to the task; 2 when there is hardly any evidence of the criterion beyond isolated words.`

// bandDescriptors are the public IELTS Writing band descriptors (Academic,
// revised May 2023) for bands 9 down to 3, condensed only by dropping the
// General Training lines. The full range anchors the model's scale at both
// ends: with only the middle bands it pulls weak answers up to 5 and holds
// strong ones down at 8.
var bandDescriptors = map[string]string{
	criterionTaskAchievement: `
  9: All the requirements of the task are fully and appropriately satisfied. There may be extremely rare lapses in content.
  8: The response covers all the requirements of the task appropriately, relevantly and sufficiently. Key features are skilfully selected, and clearly presented, highlighted and illustrated. There may be occasional omissions or lapses in content.
  7: The response covers the requirements of the task. The content is relevant and accurate; there may be a few omissions or lapses. The format is appropriate. Key features which are selected are covered and clearly highlighted but could be more fully or more appropriately illustrated or extended. It presents a clear overview, the data are appropriately categorised, and main trends or differences are identified.
  6: The response focuses on the requirements of the task and an appropriate format is used. Key features which are selected are covered and adequately highlighted. A relevant overview is attempted. Information is appropriately selected and supported using figures/data. Some irrelevant, inappropriate or inaccurate information may occur in areas of detail or when illustrating or extending the main points. Some details may be missing (or excessive) and further extension or illustration may be needed.
  5: The response generally addresses the requirements of the task. The format may be inappropriate in places. Key features which are selected are not adequately covered. The recounting of detail is mainly mechanical. There may be no data to support the description. There may be a tendency to focus on details without referring to the bigger picture. The inclusion of irrelevant, inappropriate or inaccurate material in key areas detracts from the task achievement. There is limited detail when extending and illustrating the main points.
  4: The response is an attempt to address the task. Few key features have been selected. The format may be inappropriate. Key features which are presented may be irrelevant, repetitive, inaccurate or inappropriate.
  3: The response does not address the requirements of the task, possibly because of a misunderstanding of the data or diagram. Key features which are presented may be largely irrelevant. Limited information is presented, and this may be used repetitively.`,
	criterionTaskResponse: `
  9: The prompt is appropriately addressed and explored in depth. A clear and fully developed position is presented which directly answers the question/s. Ideas are relevant, fully extended and well supported. Any lapses in content or support are extremely rare.
  8: The prompt is appropriately and sufficiently addressed. A clear and well-developed position is presented in response to the question/s. Ideas are relevant, well extended and supported. There may be occasional omissions or lapses in content.
  7: The main parts of the prompt are appropriately addressed. A clear and developed position is presented. Main ideas are extended and supported but there may be a tendency to over-generalise or there may be a lack of focus and precision in supporting ideas/material.
  6: The main parts of the prompt are addressed, though some may be more fully covered than others. An appropriate format is used. A position is presented that is directly relevant to the prompt, although the conclusions drawn may be unclear, unjustified or repetitive. Main ideas are relevant, but some may be insufficiently developed or may lack clarity, while some supporting arguments and evidence may be less relevant or inadequate.
  5: The main parts of the prompt are incompletely addressed. The format may be inappropriate in places. The writer expresses a position, but the development is not always clear. Some main ideas are put forward, but they are limited and are not sufficiently developed and/or there may be irrelevant detail. There may be some repetition.
  4: The prompt is tackled in a minimal way, or the answer is tangential, possibly due to some misunderstanding of the prompt. The format may be inappropriate. A position is discernible, but the reader has to read carefully to find it. Main ideas are difficult to identify and such ideas that are identifiable may lack relevance, clarity and/or support. Large parts of the response may be repetitive.
  3: No part of the prompt is adequately addressed, or the prompt has been misunderstood. No relevant position can be identified, and/or there is little direct response to the question/s. There are few ideas, and these may be irrelevant or insufficiently developed.`,
	criterionCoherence: `
  9: The message can be followed effortlessly. Cohesion is used in such a way that it very rarely attracts attention. Any lapses in coherence or cohesion are minimal. Paragraphing is skilfully managed.
  8: The message can be followed with ease. Information and ideas are logically sequenced, and cohesion is well managed. Occasional lapses in coherence and cohesion may occur. Paragraphing is used sufficiently and appropriately.
  7: Information and ideas are logically organised, and there is a clear progression throughout the response; a few minor lapses may occur. A range of cohesive devices including reference and substitution is used flexibly but with some inaccuracies or some over/under use. Paragraphing is generally used effectively to support overall coherence, and the sequencing of ideas within a paragraph is generally logical.
  6: Information and ideas are generally arranged coherently and there is a clear overall progression. Cohesive devices are used to some good effect but cohesion within and/or between sentences may be faulty or mechanical due to misuse, overuse or omission. The use of reference and substitution may lack flexibility or clarity and result in some repetition or error. Paragraphing may not always be logical and/or the central topic may not always be clear.
  5: Organisation is evident but is not wholly logical and there may be a lack of overall progression. Nevertheless, there is a sense of underlying coherence to the response. The relationship of ideas can be followed but the sentences are not fluently linked to each other. There may be limited/overuse of cohesive devices with some inaccuracy. The writing may be repetitive due to inadequate and/or inaccurate use of reference and substitution. Paragraphing may not be used, may be inadequate or illogical.
  4: Information and ideas are evident but not arranged coherently and there is no clear progression within the response. Relationships between ideas can be unclear and/or inadequately marked. There is some use of basic cohesive devices, which may be inaccurate or repetitive. There is inaccurate use or a lack of substitution or referencing. There may be no paragraphing and/or no clear main topic within paragraphs.
  3: There is no apparent logical organisation. Ideas are discernible but difficult to relate to each other. There is minimal use of sequencers or cohesive devices, and those used do not necessarily indicate a logical relationship between ideas. There is difficulty in identifying referencing. Any attempts at paragraphing are unhelpful.`,
	criterionLexical: `
  9: Full flexibility and precise use are evident within the scope of the task. A wide range of vocabulary is used accurately and appropriately with very natural and sophisticated control of lexical features. Minor errors in spelling and word formation are extremely rare and have minimal impact on communication.
  8: A wide resource is fluently and flexibly used to convey precise meanings. There is skilful use of uncommon and/or idiomatic items when appropriate, despite occasional inaccuracies in word choice and collocation. Occasional errors in spelling and/or word formation may occur, but have minimal impact on communication.
  7: The resource is sufficient to allow some flexibility and precision. There is some ability to use less common and/or idiomatic items. An awareness of style and collocation is evident, though inappropriacies occur. There are only a few errors in spelling and/or word formation and they do not detract from overall clarity.
  6: The resource is generally adequate and appropriate for the task. The meaning is generally clear in spite of a rather restricted range or a lack of precision in word choice. If the writer is a risk-taker, there will be a wider range of vocabulary used but higher degrees of inaccuracy or inappropriacy. There are some errors in spelling and/or word formation, but these do not impede communication.
  5: The resource is limited but minimally adequate for the task. Simple vocabulary may be used accurately but the range does not permit much variation in expression. There may be frequent lapses in the appropriacy of word choice and a lack of flexibility is apparent in frequent simplifications and/or repetitions. Errors in spelling and/or word formation may be noticeable and may cause some difficulty for the reader.
  4: The resource is limited and inadequate for or unrelated to the task. Vocabulary is basic and may be used repetitively. There may be inappropriate use of lexical chunks (memorised phrases, formulaic language and/or language from the input material). Inappropriate word choice and/or errors in word formation and/or in spelling may impede meaning.
  3: The resource is inadequate, which may be due to the response being significantly underlength. Possible over-dependence on input material or memorised language. Control of word choice and/or spelling is very limited, and errors predominate. These errors may severely impede meaning.`,
	criterionGrammar: `
  9: A wide range of structures is used with full flexibility and control. Punctuation and grammar are used appropriately throughout. Minor errors are extremely rare and have minimal impact on communication.
  8: A wide range of structures is flexibly and accurately used. The majority of sentences are error-free, and punctuation is well managed. Occasional, non-systematic errors and inappropriacies occur, but have minimal impact on communication.
  7: A variety of complex structures is used with some flexibility and accuracy. Grammar and punctuation are generally well controlled, and error-free sentences are frequent. A few errors in grammar may persist, but these do not impede communication.
  6: A mix of simple and complex sentence forms is used but flexibility is limited. Examples of more complex structures are not marked by the same level of accuracy as in simple structures. Errors in grammar and punctuation occur, but rarely impede communication.
  5: The range of structures is limited and rather repetitive. Although complex sentences are attempted, they tend to be faulty, and the greatest accuracy is achieved on simple sentences. Grammatical errors may be frequent and cause some difficulty for the reader. Punctuation may be faulty.
  4: A very limited range of structures is used. Subordinate clauses are rare and simple sentences predominate. Some structures are produced accurately but grammatical errors are frequent and may impede meaning. Punctuation is often faulty or inadequate.
  3: Sentence forms are attempted, but errors in grammar and punctuation predominate, except in memorised phrases or those taken from the input material. This prevents most meaning from coming through. Length may be insufficient to provide evidence of control of sentence forms.`,
}

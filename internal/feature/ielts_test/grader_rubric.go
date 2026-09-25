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

// bandDescriptors condenses the public IELTS band descriptors for the bands
// most answers fall in (5–8). They anchor the model's scale: without them,
// scores drift towards a generous 6.5–7 regardless of the answer.
var bandDescriptors = map[string]string{
	criterionTaskAchievement: `
  8: covers all requirements; clearly presents and highlights key features; overview present.
  7: covers the requirements; clear overview of main trends/differences; key features highlighted but could be more fully extended.
  6: addresses the requirements; overview with appropriately selected information; key features adequately covered but some details irrelevant, inappropriate or inaccurate.
  5: generally addresses the task; recounts detail mechanically with no clear overview; key features inadequately covered; may focus on details.`,
	criterionTaskResponse: `
  8: sufficiently addresses all parts; well-developed response with relevant, extended and supported ideas.
  7: addresses all parts; clear position throughout; main ideas extended and supported, though may over-generalise or lack focus.
  6: addresses all parts though some more fully than others; relevant position though conclusions may be unclear or repetitive; some main ideas inadequately developed.
  5: addresses the task only partially; position expressed but development not always clear; limited, not sufficiently developed ideas; may be repetitive.`,
	criterionCoherence: `
  8: sequences information and ideas logically; manages all aspects of cohesion well; paragraphs sufficiently and appropriately.
  7: logically organises information; clear progression throughout; range of cohesive devices, though some under-/over-use.
  6: arranges information coherently with overall progression; cohesive devices effective but cohesion within/between sentences may be faulty or mechanical.
  5: some organisation but lack of overall progression; inadequate, inaccurate or over-used cohesive devices; may be repetitive; paragraphing may be inadequate.`,
	criterionLexical: `
  8: wide resource used fluently and flexibly to convey precise meanings; skilful use of less common items; rare errors in spelling/word formation.
  7: sufficient range for flexibility and precision; less common items with some awareness of style and collocation; occasional errors in word choice, spelling or formation.
  6: adequate range for the task; attempts less common vocabulary with some inaccuracy; errors in spelling/word formation that do not impede communication.
  5: limited range, minimally adequate; noticeable errors in spelling/word formation that may cause some difficulty for the reader.`,
	criterionGrammar: `
  8: wide range of structures; majority of sentences error-free; only very occasional errors or inappropriacies.
  7: variety of complex structures; frequent error-free sentences; good control of grammar and punctuation with a few errors.
  6: mix of simple and complex sentence forms; some errors in grammar and punctuation that rarely reduce communication.
  5: limited range of structures; attempts complex sentences but these tend to be less accurate; frequent errors that may cause some difficulty.`,
}

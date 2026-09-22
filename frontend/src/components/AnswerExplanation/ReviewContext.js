import { createContext, useContext } from 'react';

// ReviewContext hands the graded review's answer key (question_order ->
// {answer, explanation, evidence}, from GET /tests/{id}/answer-key) down to
// the question controls, which render an AnswerExplanation per question,
// with onLocate(order) to show the cited text (locateLabel names where:
// the passage or the transcript) and, for listening, onListen(order) to
// replay that moment. Outside a review there's no provider and the
// explanations render nothing.
export const ReviewContext = createContext({ answerKey: null, onLocate: null, onListen: null, locateLabel: null });

export function useReview() {
  return useContext(ReviewContext);
}

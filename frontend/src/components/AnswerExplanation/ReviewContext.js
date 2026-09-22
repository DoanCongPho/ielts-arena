import { createContext, useContext } from 'react';

// ReviewContext hands the graded review's answer key (question_order ->
// {answer, explanation, evidence}, from GET /tests/{id}/answer-key) and an
// onLocate(order) callback down to the question controls, which render an
// AnswerExplanation per question. Outside a review there's no provider and
// the explanations render nothing.
export const ReviewContext = createContext({ answerKey: null, onLocate: null });

export function useReview() {
  return useContext(ReviewContext);
}

package grading

import "strings"

type CoherenceMetrics struct {
	DiscourseMarkers map[string]int `json:"discourse_markers"`
	DistinctMarkers  int            `json:"distinct_markers"`
	// MostUsedMarkerShare is the share of all marker uses taken by the
	// most frequent one — high values are the "overuse" the band 5
	// descriptor talks about.
	MostUsedMarkerShare float64 `json:"most_used_marker_share"`
	// MeanAnswerWords is the mean answer length per part ("1", "2", "3").
	MeanAnswerWords map[string]float64 `json:"mean_answer_words"`
	// MinimalAnswers counts Part 1/3 answers under 12 words — answers that
	// don't develop the topic at all.
	MinimalAnswers int `json:"minimal_answers"`
}

var discourseMarkers = []string{
	"actually", "anyway", "as a result", "as well as", "at the same time", "basically",
	"because", "but", "consequently", "for example", "for instance", "firstly", "secondly",
	"finally", "however", "i mean", "in addition", "in contrast", "in fact", "in my opinion",
	"moreover", "nevertheless", "on the other hand", "on top of that", "overall", "personally",
	"so", "such as", "that said", "therefore", "to be honest", "what's more", "whereas",
	"although", "while", "apart from that", "besides", "and then", "you know", "well",
}

func coherenceMetrics(answers []answerTranscript, answerWords map[int][]int) CoherenceMetrics {
	c := CoherenceMetrics{DiscourseMarkers: map[string]int{}, MeanAnswerWords: map[string]float64{}}
	total, most := 0, 0
	for _, a := range answers {
		text := " " + strings.Join(strings.Fields(strings.ToLower(stripPunct(a.Text))), " ") + " "
		for _, m := range discourseMarkers {
			if n := strings.Count(text, " "+m+" "); n > 0 {
				c.DiscourseMarkers[m] += n
				total += n
			}
		}
	}
	for _, n := range c.DiscourseMarkers {
		most = max(most, n)
	}
	c.DistinctMarkers = len(c.DiscourseMarkers)
	if total > 0 {
		c.MostUsedMarkerShare = round2(float64(most) / float64(total))
	}
	for part, counts := range answerWords {
		c.MeanAnswerWords[partKey(part)] = round1(meanInts(counts))
		if part == 1 || part == 3 {
			for _, n := range counts {
				if n < 12 {
					c.MinimalAnswers++
				}
			}
		}
	}
	return c
}

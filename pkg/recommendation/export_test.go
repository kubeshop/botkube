package recommendation

func (s *AggregatedRunner) Recommendations() []Recommendation {
	return s.recommendations
}

func PodResourceName() string {
	return podsResourceName
}

func IngressResourceName() string {
	return ingressResourceName
}

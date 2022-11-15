package recommendation

func (s *AggregatedRunner) Recommendations() []Recommendation {
	return s.recommendations
}

func PodResourceType() string {
	return podsResourceType
}

func IngressResourceType() string {
	return ingressResourceType
}

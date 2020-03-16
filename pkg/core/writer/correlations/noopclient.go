package correlations

// NOOPClient implements CorrelationClient interface but doesn't do anything with the correlation
type NOOPClient struct{}

func (*NOOPClient) Correlate(*Correlation)                                                         {}
func (*NOOPClient) Delete(*Correlation)                                                            {}
func (*NOOPClient) Get(dimName string, dimValue string, callback func(map[string][]string, error)) {}
func (*NOOPClient) Start()                                                                         {}

var _ CorrelationClient = &NOOPClient{}

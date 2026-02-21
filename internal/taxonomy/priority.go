package taxonomy

// TierOf returns the priority tier for a given side effect type.
func TierOf(t SideEffectType) Tier {
	tier, ok := tierMap[t]
	if !ok {
		return TierP4 // unknown types default to lowest priority
	}
	return tier
}

var tierMap = map[SideEffectType]Tier{
	// P0
	ReturnValue:        TierP0,
	ErrorReturn:        TierP0,
	SentinelError:      TierP0,
	ReceiverMutation:   TierP0,
	PointerArgMutation: TierP0,

	// P1
	SliceMutation:          TierP1,
	MapMutation:            TierP1,
	GlobalMutation:         TierP1,
	WriterOutput:           TierP1,
	HTTPResponseWrite:      TierP1,
	ChannelSend:            TierP1,
	ChannelClose:           TierP1,
	DeferredReturnMutation: TierP1,

	// P2
	FileSystemWrite:     TierP2,
	FileSystemDelete:    TierP2,
	FileSystemMeta:      TierP2,
	DatabaseWrite:       TierP2,
	DatabaseTransaction: TierP2,
	GoroutineSpawn:      TierP2,
	Panic:               TierP2,
	CallbackInvocation:  TierP2,
	LogWrite:            TierP2,
	ContextCancellation: TierP2,

	// P3
	StdoutWrite:     TierP3,
	StderrWrite:     TierP3,
	EnvVarMutation:  TierP3,
	MutexOp:         TierP3,
	WaitGroupOp:     TierP3,
	AtomicOp:        TierP3,
	TimeDependency:  TierP3,
	ProcessExit:     TierP3,
	RecoverBehavior: TierP3,

	// P4
	ReflectionMutation:     TierP4,
	UnsafeMutation:         TierP4,
	CgoCall:                TierP4,
	FinalizerRegistration:  TierP4,
	SyncPoolOp:             TierP4,
	ClosureCaptureMutation: TierP4,
}

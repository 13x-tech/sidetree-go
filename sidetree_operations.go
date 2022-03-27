package sidetree

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// TODO: Refactor this to be generalized amongst different anchoring systems
// AnchorSystemPoint is a string in the format:
// <height>:<block hash>:<tx index>:<tx hash>:<tx out index>
func NewSideTreeOp(anchorString string, height int, blockHash string, txIndex int, txHash string, txOutIndex int) SideTreeOp {
	anchorPoint := fmt.Sprintf("%d:%s:%d:%s:%d", height, blockHash, txIndex, txHash, txOutIndex)
	return SideTreeOp{
		SystemAnchorPoint: anchorPoint,
		AnchorString:      anchorString,
		Processed:         false,
	}
}

type SideTreeOp struct {
	SystemAnchorPoint string
	AnchorString      string
	Processed         bool
}

func (s *SideTreeOp) CID() string {
	parts := strings.Split(s.AnchorString, ".")

	if len(parts) < 2 {
		return ""
	}

	return parts[1]
}

func (s *SideTreeOp) Operations() int {
	parts := strings.Split(s.AnchorString, ".")

	if len(parts) < 2 {
		return 0
	}

	operations, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}

	return operations
}

func (s *SideTreeOp) Height() int {
	parts := strings.Split(s.SystemAnchorPoint, ":")

	if len(parts) < 5 {
		// Zero should be considered an invalid height, I don't want to include an error here yet
		// When this is generalized I'll come up with a better format
		return 0
	}

	height, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}

	return height
}

func (s *SideTreeOp) BlockHash() string {
	parts := strings.Split(s.SystemAnchorPoint, ":")

	if len(parts) < 5 {
		return ""
	}

	return parts[1]
}

func (s *SideTreeOp) TxIndex() int {
	parts := strings.Split(s.SystemAnchorPoint, ":")

	if len(parts) < 5 {
		return 0
	}

	txIndex, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0
	}

	return txIndex
}

func (s *SideTreeOp) TxHash() string {
	parts := strings.Split(s.SystemAnchorPoint, ":")

	if len(parts) < 5 {
		return ""
	}

	return parts[3]
}

func (s *SideTreeOp) TxOutIndex() int {
	parts := strings.Split(s.SystemAnchorPoint, ":")

	if len(parts) < 5 {
		return 0
	}

	txOutIndex, err := strconv.Atoi(parts[4])
	if err != nil {
		return 0
	}

	return txOutIndex
}

//TODO Needs Test
func SortSidetreeOps(ops []SideTreeOp) {
	sort.Slice(ops, func(i, j int) bool {
		var sortedByHeight, sortedByTxIndex, sortedByTxOutIndex bool
		sortedByHeight = ops[i].Height() < ops[j].Height()

		if ops[i].Height() == ops[j].Height() && ops[i].TxIndex() != ops[j].TxIndex() {
			sortedByTxIndex = ops[i].TxIndex() < ops[j].TxIndex()
			return sortedByTxIndex
		}

		if ops[i].Height() == ops[j].Height() && ops[i].TxIndex() == ops[j].TxIndex() {
			sortedByTxOutIndex = ops[i].TxOutIndex() < ops[j].TxOutIndex()
			return sortedByTxOutIndex
		}

		return sortedByHeight
	})
}

func OpAlreadyExists(ops []SideTreeOp, op SideTreeOp) bool {
	for _, o := range ops {
		if o.SystemAnchorPoint == op.SystemAnchorPoint {
			return true
		}
	}

	return false
}

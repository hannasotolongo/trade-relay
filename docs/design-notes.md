Copy Trade allocation = 
Signal quantity: 100 shares
Follower allocation: 25%
Result: 25 shares
100 × 2500 / 10000 = 25

quantity := signal.Quantity * account.AllocationBps / 10000
- fractional results will be rounded down (25% of 3 shares will be 0 not 0.75)
- integer divison

 * Multi account allocation
   
                Strategy Signal
                    |
              BUY 100 AAPL
                    |
          +---------+---------+
          |         |         |
          v         v         v
      Account A  Account B  Account C
         50%        25%       10%
          |         |         |
          v         v         v
       50 shares  25 shares  10 shares
   Question- should one bad follower fail entire execution or only that follower

One follower failing allocation should not automatically prevent every other valid follower from receiving an allocation.
BUY 100 AAPL

Account A — active, 50%     → 50 shares
Account B — disabled        → skipped
Account C — active, 25%     → 25 shares
Account D — allocation tiny → skipped
______________________________________________________________________________
type Failure struct {
	AccountID string
	Err       error
}

type Result struct {
	Allocations []Allocation
	Failures    []Failure
}

Why this structure? If 100 follower accounts are involved and one is disabled, blindly returning one global error could discard 99 otherwise valid allocation decisions. ignoring the disabled account would make the system difficult to audit. So we're modeling both outcomes 

-  each follower is evaluated independently. A disabled or too small account becomes a recorded failure, while valid followers still receive allocations.

**execution/order state**
CREATED
   ↓
SUBMITTING
   ↓
ACKNOWLEDGED
   ↓
PARTIALLY_FILLED
   ↓
FILLED

OrderCreated
OrderSubmitting
OrderAcknowledged
OrderPartiallyFilled
OrderFilled
OrderRejected
OrderUnknown

Example: 
Order ID:       order-123
Symbol:         AAPL
Quantity:       50
FilledQuantity: 20
Status:         PARTIALLY_FILLED
- we wanted 50 shares but only filled 20
- so CanTransitionOrder()
- will allow order not to fill
- CREATED → SUBMITTING ✓
CREATED → FILLED     ✗
CREATED → REJECTED   ✗
- an order being submitted will have three outcomes
-                     → ACKNOWLEDGED
                   /
SUBMITTING --------→ REJECTED
                   \
                    → UNKNOWN

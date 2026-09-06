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

_____________________________________________________________________________________________________________
**Important Notes and points about system below** ___________________________________________________________________________________________________________
					
**Pessimistic** =  two users try to change data at the same time leading to conflict (prevent issue upfront) use locking
- high conflict areas
- cause possible blocking
- locks cost money
  
  			VS
  
**Optimistic** = assumes conflict between users are rare, will let many users read and change data freely without locking anything
- will check for conflict only at commit time 
- allows higher concurrency and throughput
- lots of rollbacks if error occur
- need good version tracking and timestamp to see when problem occurred

**Locking**
- allows no change of value
- other end needs to wait
- expensive

**Database Isolation Levels and Concurrency**
SQL three phenomenon
1. Dirty reads: uncommitted dependency occurs when a transaction retrieves a row that has been updated by another transaction that is not yet committed. We only want to apply data base changes when we commit transaction- dirty read violates this 

2. Non repeatable reads: Occurs when transaction retrieves a row twice and that row is updated by another transaction that is committed in between.
   
3. phantom reads: Occurs when a transaction retrieves a set of rows twice and new rows are inserted into or removed from set by another transaction that is committed in between.
   - extra row

**Data base isolation**
1.Read Uncommitted 
- no isolation, any changes from outside is visible to transaction
- can get all phenomenon
  
2. Read Committed
- each query in a transaction only sees committed stuff at the time of the query
- wont get dirty read
  
3. Repeatable Read
- each query in transaction only sees committed updates at the beginning of the transaction
- shared lock
- subject to phantom read
  
4. Serializable
- slowest
- nothing runs in parallel
- wont get any issues
  
**ACID/how it works into Trade Relay**
1. Atomicity: all operations within a transaction happen or none happen at all
   - If one fails all should rollback
     
3. consistency: data must obtain rules before and after transaction


4. Isolation: concurrent transactions should not corrupt each others work
  - can my inflight transaction see changes made by other transactions?


5. Durability: once committed data survives crashes  

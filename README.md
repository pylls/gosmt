# gosmt
A golang implementation of a sparse Merkle tree with efficient caching
strategies. Note that this is a proof-of-concept.

#### Sparse Merkle Trees
A sparse Merkle tree (SMT) is a Merkle (hash) tree that contains a leaf
for every possible output of a hash function
[[0]](http://www.links.org/files/RevocationTransparency.pdf).
In other words, a SMT has
2^N leafs for a hash function with a N-bit output, and for example when
using SHA-256 this means 2^256 leafs.
Since the full in-memory representation of an SMT is impractical (to say the
least) we have to simulate it, and it turns out that simulation is
practical. This is because the tree is _sparse_: most leafs are empty, so
when we calculate the hash of the empty leafs we get the same hash. The same
is true for interior nodes whose children are all empty, and so on.  

#### Caching Strategies
While the concept of a SMT is neat on its own, it gets better.
When simulating a SMT we can
keep a _cache_ of previously calculated nodes in the SMT. The most obvious part
is for all sparse nodes. Beyond that, there are a number of caching strategies
that give different space-time trade-offs. One cool strategy is to make the
cache probabilistic and smaller than the data to authenticate: let's say you
have a database of n items, it turns out that only keeping a cache of size 0.6*n
is practical for many applications.

Below is a graph of the size of the authenticated data structure as a function
of the size of the underlying data structure (the database to authenticate).
We include a hash treap (HT) in our comparison: good representation of
related authenticated data structures that are explicity stored in memory
[[1]](http://tamperevident.cs.rice.edu/papers/techreport-padbench.pdf).
For a SMT you have three caching stragies: B, B+, and B-0.5. The B cache stores
all (non-default) branches in the tree, B+ all children of all branches in the
tree, and B-0.5 stores 50% of all branches in the tree.
As we can see, the size of the HT is roughly eight times larger than that of
the B-0.5 cache.

![cache sizes](https://raw.github.com/pylls/gosmt/master/doc/cachesize.png)

There is no such thing as a free lunch though. Below is the average time it
takes to generate an (Merkle) audit path. We include a number of B- caches with
different caching probabilities (note that the B cache is identical to B-1.0).
While B-0.5 behaves erratic, B-0.6 and above needs less then 4ms.

![auditpath time](https://raw.github.com/pylls/gosmt/master/doc/auditpathgen.png)

You can reproduce these benchmarks with the 
[cmd/benchht](https://github.com/pylls/gosmt/tree/master/cmd/benchht) and
[cmd/benchsmt](https://github.com/pylls/gosmt/tree/master/cmd/benchsmt)
executables.

#### Paper
TODO

#### License
Apache 2.0


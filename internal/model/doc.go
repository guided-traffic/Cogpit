// Package model holds Cogpit's resource model: object references, kind profiles, table rows,
// status tokens and detail sections.
//
// Layer L0: model does no I/O and imports no client-go transport package. It takes unstructured
// content and returns values, so every function here is testable and fuzzable without a cluster.
// Unstructured content is hostile input: helpers never panic on unexpected shapes.
package model

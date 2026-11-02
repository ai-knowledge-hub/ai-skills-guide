# Retired offline compatibility prototype

The offline evaluator introduced by PR #88 has been removed after a consumer
audit. It was isolated from the website, registry, installer and runtime paths.
The owner selected maintainer-updated version information for the static site;
signed observations and a verification service are not prerequisites for it.

Use [the maintainer guide](maintaining-compatibility.md) and
[the catalogue contract](../shared/contracts/catalogue-compatibility-v1.md).
The prior implementation and evaluation design remain recoverable in PR #88
and Git history. Existing runtime credential and age checks were not changed
by removing the prototype.

import 'package:characters/characters.dart';

final class FuzzyMatch {
  const FuzzyMatch(this.indices);

  final List<int> indices;
}

FuzzyMatch? fuzzySubsequenceMatch(String candidate, String query) {
  final needle = query.trim();
  if (needle.isEmpty) return const FuzzyMatch([]);

  final candidateCharacters = candidate.characters.toList(growable: false);
  final queryCharacters = needle.characters.toList(growable: false);
  final indices = <int>[];
  var queryIndex = 0;

  for (
    var candidateIndex = 0;
    candidateIndex < candidateCharacters.length &&
        queryIndex < queryCharacters.length;
    candidateIndex += 1
  ) {
    if (candidateCharacters[candidateIndex].toLowerCase() ==
        queryCharacters[queryIndex].toLowerCase()) {
      indices.add(candidateIndex);
      queryIndex += 1;
    }
  }

  return queryIndex == queryCharacters.length ? FuzzyMatch(indices) : null;
}

bool fuzzyMatches(String candidate, String query) =>
    fuzzySubsequenceMatch(candidate, query) != null;

bool fuzzyMatchesAny(Iterable<String> candidates, String query) {
  final needle = query.trim();
  return needle.isEmpty ||
      candidates.any((candidate) => fuzzyMatches(candidate, needle));
}

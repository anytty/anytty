import 'dart:io';

Uri? resolveBrowserNavigation(String input) {
  final value = input.trim();
  if (value.startsWith('about:') || value.contains('://')) {
    return Uri.tryParse(value);
  }
  if (!value.contains(RegExp(r'\s'))) {
    // A bare IPv6 literal needs brackets before it can be parsed as a URL.
    final literal = InternetAddress.tryParse(value);
    final authority = literal?.type == InternetAddressType.IPv6
        ? '[$value]'
        : value;
    final address = Uri.tryParse('https://$authority');
    final host = address?.host ?? '';
    final bareHost = host.startsWith('[') && host.endsWith(']')
        ? host.substring(1, host.length - 1)
        : host;
    if (InternetAddress.tryParse(bareHost) != null || bareHost == 'localhost') {
      return address!.replace(scheme: 'http');
    }
    if (host.contains('.')) return address;
  }
  return Uri.https('www.google.com', '/search', {'q': value});
}

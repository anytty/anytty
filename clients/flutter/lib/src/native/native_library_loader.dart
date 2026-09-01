import 'dart:ffi';
import 'dart:io';

DynamicLibrary loadAnyttyLibrary(String androidName) {
  if (Platform.isAndroid) return DynamicLibrary.open(androidName);
  if (Platform.isIOS) return DynamicLibrary.process();
  throw UnsupportedError('AnyTTY native libraries require Android or iOS');
}

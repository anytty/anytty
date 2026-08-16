import type { CapacitorConfig } from '@capacitor/cli'

const config: CapacitorConfig = {
  appId: 'com.anytty.app',
  appName: 'AnyTTY',
  webDir: 'dist',
  // Native bridge responses contain an ephemeral loopback bearer token. Capacitor framework logs
  // must never echo plugin payloads; product diagnostics use the explicitly redacted debug log.
  loggingBehavior: 'none',
  server: {
    hostname: 'localhost',
    appStartPath: '/index.html',
    androidScheme: 'http',
    allowNavigation: [],
  },
  plugins: {
    Keyboard: {
      resize: 'none',
    },
    StatusBar: {
      backgroundColor: '#000000',
      style: 'DARK',
    },
  },
  android: {
    allowMixedContent: false,
  },
}

export default config

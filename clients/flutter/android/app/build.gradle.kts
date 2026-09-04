plugins {
    id("com.android.application")
    // The Flutter Gradle Plugin must be applied after the Android and Kotlin Gradle plugins.
    id("dev.flutter.flutter-gradle-plugin")
}

android {
    namespace = "com.anytty.app"
    compileSdk = flutter.compileSdkVersion
    ndkVersion = "28.2.13676358"

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    defaultConfig {
        applicationId = "com.anytty.app"
        // You can update the following values to match your application needs.
        // For more information, see: https://flutter.dev/to/review-gradle-config.
        minSdk = 24
        targetSdk = flutter.targetSdkVersion
        // Uses the version code from pubspec.yaml. When using split APKs, 1000 * ABI_VERSION
        // is added automatically by Flutter. (https://developer.android.com/studio/build/configure-apk-splits#configure-APK-versions)
        // You can force using the value of versionCode by specifying the `-P force-version-code-ignoring-abi=true`
        // flag during build.
        versionCode = flutter.versionCode
        versionName = flutter.versionName
    }

    val uploadStoreFile = System.getenv("ANYTTY_ANDROID_UPLOAD_STORE_FILE")
    val uploadStorePassword = System.getenv("ANYTTY_ANDROID_UPLOAD_STORE_PASSWORD")
    val uploadKeyAlias = System.getenv("ANYTTY_ANDROID_UPLOAD_KEY_ALIAS")
    val uploadKeyPassword = System.getenv("ANYTTY_ANDROID_UPLOAD_KEY_PASSWORD")
    val hasUploadSigning = listOf(
        uploadStoreFile,
        uploadStorePassword,
        uploadKeyAlias,
        uploadKeyPassword,
    ).all { !it.isNullOrBlank() }

    signingConfigs {
        getByName("debug") {
            // Keep sideloaded debug builds installable on older OEM installers.
            isV1SigningEnabled = true
            isV2SigningEnabled = true
        }
        if (hasUploadSigning) {
            create("upload") {
                storeFile = file(uploadStoreFile!!)
                storePassword = uploadStorePassword
                keyAlias = uploadKeyAlias
                keyPassword = uploadKeyPassword
                isV1SigningEnabled = true
                isV2SigningEnabled = true
            }
        }
    }

    buildTypes {
        release {
            signingConfig = if (hasUploadSigning) {
                signingConfigs.getByName("upload")
            } else {
                signingConfigs.getByName("debug")
            }
        }
    }
}

kotlin {
    compilerOptions {
        jvmTarget = org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_17
    }
}

flutter {
    source = "../.."
}

dependencies {
    implementation("androidx.webkit:webkit:1.15.0")
}

val anyttyRepoRoot = rootProject.projectDir.resolve("../../..").canonicalFile
val buildAnyttyNative by tasks.registering(Exec::class) {
    group = "build"
    description = "Builds the AnyTTY Go client and Ghostty terminal input libraries."
    workingDir(anyttyRepoRoot)
    commandLine(
        "bash",
        anyttyRepoRoot.resolve("scripts/build-flutter-android-native.sh"),
        projectDir.resolve("src/main/jniLibs"),
    )
}

tasks.named("preBuild") {
    dependsOn(buildAnyttyNative)
}

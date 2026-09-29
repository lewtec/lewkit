plugins {
    id("com.android.application")
}

android {
    namespace = "{{.PackageID}}"
    compileSdk = 35

    defaultConfig {
        applicationId = "{{.PackageID}}"
        minSdk = 26
        targetSdk = 35
        versionCode = {{.VersionCode}}
        versionName = "{{.VersionName}}"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    // Single-APK packaging; ABIs come from jniLibs/ (default: arm64-v8a).
    splits {
        abi {
            isEnable = false
        }
    }
    // Extract jniLibs to the filesystem so ProcessBuilder can exec libeletrocromo.so
    // (filesDir is often noexec; nativeLibraryDir must be a real extracted path).
    packaging {
        jniLibs {
            useLegacyPackaging = true
        }
    }
}

// AndroidX pulls kotlin-stdlib 1.8 and an older jdk8 jar. Align them,
// or the duplicate-class check fails without the Kotlin Gradle plugin.
configurations.configureEach {
    resolutionStrategy.eachDependency {
        if (requested.group == "org.jetbrains.kotlin") {
            useVersion("1.8.22")
        }
    }
}

dependencies {
    implementation("androidx.core:core:1.15.0")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.webkit:webkit:1.12.1")
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
}

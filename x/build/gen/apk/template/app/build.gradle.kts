plugins {
    id("com.android.application")
    id("org.jetbrains.kotlin.android")
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
    kotlinOptions {
        jvmTarget = "17"
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

dependencies {
    implementation("androidx.core:core-ktx:1.15.0")
    implementation("androidx.appcompat:appcompat:1.7.0")
    implementation("androidx.webkit:webkit:1.12.1")
    implementation("androidx.swiperefreshlayout:swiperefreshlayout:1.1.0")
}

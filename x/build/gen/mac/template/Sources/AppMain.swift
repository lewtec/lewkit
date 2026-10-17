import Darwin
import Foundation

/// Replaces this process with the Go program beside it.
/// That program opens app.GUI and app.Web. This file does not open a window.
@main
enum AppMain {
    private static let programName = "eletrocromo-server"

    static func main() {
        let program = Bundle.main.bundleURL
            .appendingPathComponent("Contents/MacOS")
            .appendingPathComponent(programName)
        guard FileManager.default.isExecutableFile(atPath: program.path) else {
            fputs("missing \(programName) in the app bundle\n", stderr)
            Darwin.exit(1)
        }
        var args = CommandLine.arguments
        args[0] = program.path
        let count = args.count
        let argv = UnsafeMutablePointer<UnsafeMutablePointer<CChar>?>.allocate(capacity: count + 1)
        for (index, arg) in args.enumerated() {
            argv[index] = strdup(arg)
        }
        argv[count] = nil
        Darwin.execv(program.path, argv)
        fputs("exec \(programName) failed\n", stderr)
        Darwin.exit(1)
    }
}

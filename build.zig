// ~/.GH/Qompass/Go/beacon/build.zig
// ---------------------------------
// Copyright (C) 2025 Qompass AI, All rights reserved

const std = @import("std");

pub fn build(b: *std.Build) !void {
    const target = b.standardTargetOptions(.{});
    const optimize = b.standardOptimizeOption(.{});

    const zig_lib = b.addStaticLibrary(.{
        .name = "beacon-zig",
        .root_source_file = .{ .path = "src/main.zig" },
        .target = target,
        .optimize = optimize,
    });
    zig_lib.linkLibC();

    const go_build = b.addSystemCommand(&[_][]const u8{
        "go", "build",
        "-trimpath",
        "-o", "bin/beacon",
        "-tags", "release",
    });

    const targets = .{
        .{ .triple = "x86_64-linux-musl" },
        .{ .triple = "aarch64-linux-musl" },
        .{ .triple = "x86_64-macos" },
        .{ .triple = "aarch64-macos" },
        .{ .triple = "x86_64-windows" },
    };

    for (targets) |t| {
        const target_str = t.triple;
        const cross_target = std.zig.CrossTarget.parse(.{ .arch_os_abi = target_str }) catch unreachable;

        const cross_lib = b.addStaticLibrary(.{
            .name = "beacon-zig",
            .root_source_file = .{ .path = "src/main.zig" },
            .target = cross_target,
            .optimize = optimize,
        });
        cross_lib.linkLibC();

        const cross_step = b.addSystemCommand(&[_][]const u8{
            "go", "build",
            "-trimpath",
            "-o", b.fmt("bin/beacon-{s}", .{target_str}),
            "-tags", "release",
        });

        cross_step.setEnvironmentVariable("CGO_ENABLED", "1");
        cross_step.setEnvironmentVariable("CC", b.fmt("zig cc -target {s}", .{target_str}));
        cross_step.setEnvironmentVariable("CXX", b.fmt("zig c++ -target {s}", .{target_str}));
        
        cross_step.addArgs(&.{
            "-ldflags", 
            b.fmt("-extldflags '-L{d} -lbeacon-zig'", .{cross_lib.getOutputDirectory()}),
        });
        
        cross_step.step.dependOn(&cross_lib.step);
        
        const install


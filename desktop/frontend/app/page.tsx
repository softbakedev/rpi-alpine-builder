"use client"

import React from "react"
import { AlpineConfigForm } from "../components/AlpineConfigForm"
import { ThemeProvider } from "../components/ThemeProvider"
import { useAlpineVersions, useVolumeDirectories, useWifiNetworks } from "../lib/api"

export default function Home() {
    const { data: versions, isLoading: versionsLoading, isError: versionsError } = useAlpineVersions()
    const { data: directories, isLoading: directoriesLoading, isError: directoriesError } = useVolumeDirectories()
    // const { data: networks, isLoading: networksLoading, isError: networksError } = useWifiNetworks()

    return (
        <ThemeProvider>
            <div className="min-h-screen bg-gray-100 dark:bg-gray-900 text-gray-900 dark:text-white">
                <div className="container mx-auto px-4 py-8">
                    <h1 className="text-4xl font-bold text-center mb-8 text-[#e30b5d]">Raspberry Pi Alpine Image Builder</h1>
                    <AlpineConfigForm
                        versions={versions}
                        versionsLoading={versionsLoading}
                        versionsError={versionsError}
                        directories={directories}
                        directoriesLoading={directoriesLoading}
                        directoriesError={directoriesError}
                    />
                </div>
            </div>
        </ThemeProvider>
    )
}

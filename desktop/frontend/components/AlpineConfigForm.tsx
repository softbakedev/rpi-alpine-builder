"use client"

import React, { useState, useEffect } from "react"
import { useForm, Controller } from "react-hook-form"
import Image from "next/image"
import { ChevronDownIcon, CheckCircleIcon, XCircleIcon, ExclamationCircleIcon } from "@heroicons/react/24/solid"
import { GithubIcon } from "lucide-react"
import type { AlpineConfig } from "../types/config"
import { useTheme } from "./ThemeProvider"
import { BASE_URL } from '@/lib/api'

interface AlpineConfigFormProps {
    versions: string[] | null
    versionsLoading: boolean
    versionsError: boolean
    directories: string[] | null
    directoriesLoading: boolean
    directoriesError: boolean
}

// Define interfaces for structured responses
interface SuccessResponse {
    message: string;
}

interface ErrorResponse {
    error: string;
}

// Function to handle JSON error responses
const handleErrorResponse = async (response: Response): Promise<ErrorResponse> => {
    try {
        const errorData: ErrorResponse = await response.json();
        return errorData;
    } catch (parseError) {
        // If parsing fails, return a generic error message
        return { error: "An unexpected error occurred." };
    }
};

export function AlpineConfigForm({
                                     versions,
                                     versionsLoading,
                                     versionsError,
                                     directories,
                                     directoriesLoading,
                                     directoriesError,
                                 }: AlpineConfigFormProps) {
    const { isDarkMode, toggleTheme } = useTheme()
    const [buildStatus, setBuildStatus] = useState<"idle" | "building" | "success" | "error">("idle")
    const [errorMessage, setErrorMessage] = useState<string | null>(null)
    const [buildProgress, setBuildProgress] = useState(0)

    const {
        control,
        handleSubmit,
        watch,
        setValue,
        formState: { errors, isValid },
    } = useForm<AlpineConfig>({
        mode: "onChange",
        defaultValues: {
            version: "",
            hostname: "",
            wifiNetwork: "",
            wifiPassword: "",
            rootPassword: "",
            volumeDirectory: "",
        },
    })

    const onSubmit = async (data: AlpineConfig) => {
        setBuildStatus("building");
        setErrorMessage(null);
        setBuildProgress(0);

        try {
            const response = await fetch(`${BASE_URL}/api/build`, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(data),
            });

            if (!response.ok) {
                const errorData = await handleErrorResponse(response);
                throw new Error(`Build failed: ${errorData.error}`);
            }

            const reader = response.body?.getReader();
            if (!reader) {
                throw new Error("Unable to read response stream.");
            }

            const decoder = new TextDecoder();
            let done = false;

            while (!done) {
                const { value, done: doneReading } = await reader.read();
                done = doneReading;
                if (value) {
                    const chunk = decoder.decode(value, { stream: true });
                    // Assuming the server sends progress updates as plain numbers in the JSON
                    const progress = Number.parseInt(chunk, 10);
                    if (!isNaN(progress)) {
                        setBuildProgress(progress);
                    }
                }
            }

            setBuildStatus("success");
            // Optionally use successData.message if needed
        } catch (error) {
            setBuildStatus("error");
            setErrorMessage(error instanceof Error ? error.message : "An unknown error occurred.");
        }
    };

    useEffect(() => {
        if (buildStatus === "success" || buildStatus === "error") {
            const timer = setTimeout(() => {
                setBuildStatus("idle")
                setBuildProgress(0)
            }, 1000)
            return () => clearTimeout(timer)
        }
    }, [buildStatus])

    useEffect(() => {
        if (errorMessage) {
            const timer = setTimeout(() => {
                setErrorMessage(null)
            }, 5000)
            return () => clearTimeout(timer)
        }
    }, [errorMessage])

    const renderField = (name: keyof AlpineConfig, placeholder: string, type = "text", options?: string[]) => (
        <div className="relative group">
            <Controller
                name={name}
                control={control}
                rules={{ required: true }}
                render={({ field }) => (
                    <>
                        {type === "select" ? (
                            <select
                                {...field}
                                className="block w-full px-4 py-3 pr-8 text-white bg-[#e30b5d] border-2 border-white rounded-lg appearance-none focus:outline-none focus:ring-2 focus:ring-white focus:border-transparent transition-all duration-300 ease-in-out placeholder-white placeholder-opacity-75"
                            >
                                <option value="" disabled hidden>
                                    {placeholder}
                                </option>
                                {options?.map((option) => (
                                    <option key={option} value={option}>
                                        {option}
                                    </option>
                                ))}
                            </select>
                        ) : (
                            <input
                                type={type}
                                {...field}
                                placeholder={placeholder}
                                className="block w-full px-4 py-3 text-white bg-[#e30b5d] border-2 border-white rounded-lg appearance-none focus:outline-none focus:ring-2 focus:ring-white focus:border-transparent transition-all duration-300 ease-in-out placeholder-white placeholder-opacity-75"
                            />
                        )}
                        {errors[name] && (
                            <ExclamationCircleIcon className="absolute top-1/2 right-3 transform -translate-y-1/2 h-5 w-5 text-yellow-400" />
                        )}
                    </>
                )}
            />
            {type === "select" && (
                <div className="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none">
                    <ChevronDownIcon className="w-5 h-5 text-white" />
                </div>
            )}
        </div>
    )

    const renderWifiNetworkField = () => (
        <div className="relative group">
            <Controller
                name="wifiNetwork"
                control={control}
                rules={{ required: true }}
                render={({ field }) => (
                    <input
                        {...field}
                        type="text"
                        placeholder="Enter WiFi Network Name"
                        className="block w-full px-4 py-3 text-white bg-[#e30b5d] border-2 border-white rounded-lg appearance-none focus:outline-none focus:ring-2 focus:ring-white focus:border-transparent transition-all duration-300 ease-in-out placeholder-white placeholder-opacity-75"
                    />
                )}
            />
            {errors.wifiNetwork && (
                <ExclamationCircleIcon className="absolute top-1/2 right-3 transform -translate-y-1/2 h-5 w-5 text-yellow-400" />
            )}
        </div>
    )

    return (
        <>
            <div className="flex justify-center mb-8">
                <Image
                    src="https://hebbkx1anhila5yf.public.blob.vercel-storage.com/logo-5HpcX46GXCab43Pr1TKYwHsbeXDss6.png"
                    alt="Alpine Linux Logo"
                    width={120}
                    height={120}
                    priority
                    className="rounded-full shadow-lg"
                />
            </div>

            <div className="bg-[#e30b5d] shadow-2xl rounded-2xl overflow-hidden backdrop-blur-lg bg-opacity-90 max-w-4xl mx-auto">
                <form onSubmit={handleSubmit(onSubmit)} className="space-y-6 p-8 md:p-12">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        {renderField("version", "Select Alpine Version", "select", versions || [])}
                        {renderField("hostname", "Enter Hostname")}
                        {renderWifiNetworkField()}
                        {renderField("wifiPassword", "Enter WiFi Password", "password")}
                        {renderField("rootPassword", "Enter Root Password", "password")}
                        {renderField("volumeDirectory", "Select Volume Directory", "select", directories || [])}
                    </div>

                    <div className="pt-4">
                        <button
                            type="submit"
                            className={`w-full py-3 px-4 text-white font-medium rounded-lg shadow-md transition-all duration-300 ease-in-out 
                ${
                                !isValid || buildStatus === "building"
                                    ? "bg-green-400 cursor-not-allowed"
                                    :  buildStatus === "error" ? "bg-red-400 cursor-not-allowed" : 
                                        "bg-green-500 hover:bg-gradient-to-r hover:from-green-400 hover:to-green-600 active:bg-gradient-to-r active:from-green-500 active:to-green-700"
                            } focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 relative overflow-hidden`}
                            disabled={!isValid || buildStatus === "building"}
                        >
                            <div className="flex items-center justify-center">
                                {buildStatus === "building" && (
                                    <div
                                        className="absolute inset-0 bg-green-600"
                                        style={{ width: `${buildProgress}%`, transition: "width 0.3s ease-in-out" }}
                                    ></div>
                                )}
                                <span className="relative z-10">
                  {buildStatus === "building" ? (
                      "Building..."
                  ) : buildStatus === "success" ? (
                      <>
                          <CheckCircleIcon className="w-5 h-5 inline-block mr-2" />
                          Build Complete!
                      </>
                  ) : buildStatus === "error" ? (
                      <>
                          <XCircleIcon className="w-5 h-5 inline-block mr-2" />
                          Build Failed
                      </>
                  ) : (
                      "Start Build"
                  )}
                </span>
                            </div>
                        </button>
                    </div>
                </form>
            </div>

            <div className="flex justify-center items-center space-x-4 mt-8">
                <a
                    href="https://github.com/alpine-linux"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white transition-colors duration-300"
                >
                    <GithubIcon size={28} />
                </a>
                <button
                    onClick={toggleTheme}
                    className="text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white transition-colors duration-300"
                >
                    {isDarkMode ? (
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            fill="none"
                            viewBox="0 0 24 24"
                            strokeWidth={1.5}
                            stroke="currentColor"
                            className="w-6 h-6"
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                d="M12 3v2.25m6.364.386l-1.591 1.591M21 12h-2.25m-.386 6.364l-1.591-1.591M12 18.75V21m-4.773-4.227l-1.591 1.591M5.25 12H3m4.227-4.773L5.636 5.636M15.75 12a3.75 3.75 0 11-7.5 0 3.75 3.75 0 017.5 0z"
                            />
                        </svg>
                    ) : (
                        <svg
                            xmlns="http://www.w3.org/2000/svg"
                            fill="none"
                            viewBox="0 0 24 24"
                            strokeWidth={1.5}
                            stroke="currentColor"
                            className="w-6 h-6"
                        >
                            <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                d="M21.752 15.002A9.718 9.718 0 0118 15.75c-5.385 0-9.75-4.365-9.75-9.75 0-1.33.266-2.597.748-3.752A9.753 9.753 0 003 11.25C3 16.635 7.365 21 12.75 21a9.753 9.753 0 009.002-5.998z"
                            />
                        </svg>
                    )}
                </button>
            </div>

            {errorMessage && (
                <div
                    className="mt-6 bg-red-100 border-l-4 border-red-500 text-red-700 p-4 rounded-lg shadow-md transition-opacity duration-300 ease-in-out max-w-4xl mx-auto"
                    role="alert"
                    style={{ opacity: errorMessage ? 1 : 0 }}
                >
                    <div className="flex">
                        <div className="flex-shrink-0">
                            <XCircleIcon className="h-5 w-5 text-red-500" />
                        </div>
                        <div className="ml-3">
                            <p className="font-bold">Error</p>
                            <p>{errorMessage}</p>
                        </div>
                    </div>
                </div>
            )}
        </>
    )
}
"use client"

import React, { useState } from "react"
import { useForm, Controller } from "react-hook-form"
import Image from "next/image"
import {
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Box,
  Typography,
  Container,
  Paper,
  Grid,
  IconButton,
  CircularProgress,
  Snackbar,
  Alert,
} from "@mui/material"
import {
  ExpandMore as ExpandMoreIcon,
  ExpandLess as ExpandLessIcon,
  GitHub as GitHubIcon,
  DarkMode as DarkModeIcon,
  LightMode as LightModeIcon,
  Check as CheckIcon,
  Error as ErrorIcon,
} from "@mui/icons-material"
import { ThemeProvider, createTheme } from "@mui/material/styles"
import { useAlpineVersions, useVolumeDirectories, useWifiNetworks } from "./lib/api"
import type { AlpineConfig } from "./types/config"

const darkTheme = createTheme({
  palette: {
    mode: "dark",
    primary: {
      main: "#e30b5d",
    },
    secondary: {
      main: "#4caf50",
    },
  },
})

const lightTheme = createTheme({
  palette: {
    mode: "light",
    primary: {
      main: "#e30b5d",
    },
    secondary: {
      main: "#4caf50",
    },
  },
})

export default function AlpineConfig() {
  const [isOpen, setIsOpen] = useState(true)
  const [isDarkMode, setIsDarkMode] = useState(true)
  const [buildStatus, setBuildStatus] = useState<"idle" | "building" | "success" | "error">("idle")
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const {
    control,
    handleSubmit,
    watch,
    formState: { isValid },
  } = useForm<AlpineConfig>({
    mode: "onChange",
  })

  const { data: versions, isLoading: versionsLoading, isError: versionsError } = useAlpineVersions()
  const { data: directories, isLoading: directoriesLoading, isError: directoriesError } = useVolumeDirectories()
  const { data: networks, isLoading: networksLoading, isError: networksError } = useWifiNetworks()

  const onSubmit = async (data: AlpineConfig) => {
    setBuildStatus("building")
    setErrorMessage(null)

    try {
      // Simulate build process
      await new Promise((resolve) => setTimeout(resolve, 3000))

      // Simulate random success/failure
      if (Math.random() > 0.5) {
        throw new Error("Build failed: Unable to create image")
      }

      setBuildStatus("success")

      // Reset success state after 3 seconds
      setTimeout(() => {
        setBuildStatus("idle")
      }, 3000)
    } catch (error) {
      setBuildStatus("error")
      setErrorMessage(error instanceof Error ? error.message : "An unknown error occurred")
    }
  }

  const handleCloseError = () => {
    setErrorMessage(null)
  }

  return (
    <ThemeProvider theme={isDarkMode ? darkTheme : lightTheme}>
      <Container
        maxWidth="md"
        sx={{ minHeight: "100vh", display: "flex", flexDirection: "column", justifyContent: "center", py: 4 }}
      >
        <Typography variant="h4" component="h1" align="center" gutterBottom>
          Raspberry Pi Alpine Image Builder
        </Typography>

        <Box sx={{ display: "flex", justifyContent: "center", mb: 4 }}>
          <Image
            src="https://hebbkx1anhila5yf.public.blob.vercel-storage.com/logo-5HpcX46GXCab43Pr1TKYwHsbeXDss6.png"
            alt="Alpine Linux Logo"
            width={120}
            height={120}
            priority
          />
        </Box>

        <Paper elevation={3} sx={{ p: 4 }}>
          <form onSubmit={handleSubmit(onSubmit)}>
            <Button
              variant="contained"
              fullWidth
              sx={{ mb: 2, bgcolor: "#e30b5d", "&:hover": { bgcolor: "#c90950" } }}
              onClick={() => setIsOpen(!isOpen)}
              endIcon={isOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
            >
              Configure Alpine Linux
            </Button>

            {isOpen && (
              <Grid container spacing={3}>
                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Alpine Version</InputLabel>
                    <Controller
                      name="version"
                      control={control}
                      rules={{ required: true }}
                      render={({ field }) => (
                        <Select {...field} label="Alpine Version">
                          {versionsLoading ? (
                            <MenuItem disabled>Loading...</MenuItem>
                          ) : (
                            versions?.map((version: string) => (
                              <MenuItem key={version} value={version}>
                                {version}
                              </MenuItem>
                            ))
                          )}
                        </Select>
                      )}
                    />
                  </FormControl>
                </Grid>

                <Grid item xs={12} sm={6}>
                  <Controller
                    name="hostname"
                    control={control}
                    rules={{ required: true }}
                    render={({ field }) => <TextField {...field} fullWidth label="Hostname" />}
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <Controller
                      name="wifiNetwork"
                      control={control}
                      rules={{ required: true }}
                      render={({ field }) => <TextField {...field} fullWidth label="WiFi Network Name" />}
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <Controller
                    name="wifiPassword"
                    control={control}
                    rules={{ required: true }}
                    render={({ field }) => <TextField {...field} fullWidth label="WiFi Password" type="password" />}
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <Controller
                    name="rootPassword"
                    control={control}
                    rules={{ required: true }}
                    render={({ field }) => <TextField {...field} fullWidth label="Root Password" type="password" />}
                  />
                </Grid>

                <Grid item xs={12} sm={6}>
                  <FormControl fullWidth>
                    <InputLabel>Volume Directory</InputLabel>
                    <Controller
                      name="volumeDirectory"
                      control={control}
                      rules={{ required: true }}
                      render={({ field }) => (
                        <Select {...field} label="Volume Directory">
                          {directoriesLoading ? (
                            <MenuItem disabled>Loading...</MenuItem>
                          ) : (
                            directories?.map((directory: string) => (
                              <MenuItem key={directory} value={directory}>
                                {directory}
                              </MenuItem>
                            ))
                          )}
                        </Select>
                      )}
                    />
                  </FormControl>
                </Grid>
              </Grid>
            )}

            <Button
              type="submit"
              variant="contained"
              color="secondary"
              fullWidth
              sx={{ mt: 3, height: 56 }}
              disabled={!isValid || buildStatus === "building"}
            >
              {buildStatus === "building" ? (
                <CircularProgress size={24} color="inherit" />
              ) : buildStatus === "success" ? (
                <CheckIcon />
              ) : buildStatus === "error" ? (
                <ErrorIcon />
              ) : (
                "Start Build"
              )}
              {buildStatus === "building" ? " Building..." : buildStatus === "success" ? " Build Complete!" : ""}
            </Button>
          </form>
        </Paper>

        <Box sx={{ display: "flex", justifyContent: "center", alignItems: "center", mt: 4 }}>
          <IconButton
            href="https://github.com/alpine-linux"
            target="_blank"
            rel="noopener noreferrer"
            aria-label="View on GitHub"
          >
            <GitHubIcon />
          </IconButton>
          <IconButton onClick={() => setIsDarkMode(!isDarkMode)} aria-label="Toggle dark mode">
            {isDarkMode ? <LightModeIcon /> : <DarkModeIcon />}
          </IconButton>
        </Box>

        <Snackbar open={!!errorMessage} autoHideDuration={6000} onClose={handleCloseError}>
          <Alert onClose={handleCloseError} severity="error" sx={{ width: "100%" }}>
            {errorMessage}
          </Alert>
        </Snackbar>
      </Container>
    </ThemeProvider>
  )
}


import { useState, useEffect } from 'react'

export const BASE_URL = "http://localhost:8080"

const fetchData = async (url: string) => {
  try {
    const response = await fetch(`${BASE_URL}${url}`, {
      headers: {
        'Accept': 'application/json',
      },
    })
    if (!response.ok) {
      throw new Error('An error occurred while fetching the data.')
    }
    return await response.json()
  } catch (error) {
    console.error(`Error fetching data from ${url}:`, error)
    return null
  }
}

interface UseApiResult<T> {
  data: T | null
  isLoading: boolean
  isError: boolean
}

function useApi<T>(url: string): UseApiResult<T> {
  const [data, setData] = useState<T | null>(null)
  const [isLoading, setIsLoading] = useState<boolean>(true)
  const [isError, setIsError] = useState<boolean>(false)

  useEffect(() => {
    let isMounted = true
    const fetchDataAndUpdate = async () => {
      setIsLoading(true)
      const result = await fetchData(url)
      if (isMounted) {
        if (result === null) {
          setIsError(true)
        } else {
          setData(result)
          setIsError(false)
        }
        setIsLoading(false)
      }
    }

    fetchDataAndUpdate()

    const intervalId = setInterval(fetchDataAndUpdate, 10000) // Fetch every 10 seconds

    return () => {
      isMounted = false
      clearInterval(intervalId)
    }
  }, [url])

  return { data, isLoading, isError }
}

export function useAlpineVersions() {
  return useApi<string[]>("/api/alpine-versions")
}

export function useVolumeDirectories() {
  return useApi<string[]>("/api/volume-directories")
}

export function useWifiNetworks() {
  return useApi<string[]>("/api/wifi-networks")
}

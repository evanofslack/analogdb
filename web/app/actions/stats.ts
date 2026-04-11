"use server";

import { statsApi } from "@lib/client";
import {
  StatsOverviewGetRequest,
  StatsPostsOverTimeGetRequest,
  StatsFilmsGetRequest,
  StatsCamerasGetRequest,
  StatsColorsGetRequest,
  ServerStatsOverviewResponse,
  ServerStatsPeriodsResponse,
  ServerStatsFilmsResponse,
  ServerStatsCamerasResponse,
  ServerStatsColorsResponse,
} from "analogdb-generated";

export async function getStatsOverview(
  params: StatsOverviewGetRequest
): Promise<ServerStatsOverviewResponse> {
  try {
    return await statsApi.statsOverviewGet(params);
  } catch (error) {
    console.error("get stats overview failed:", error);
    throw error;
  }
}

export async function getStatsPostsOverTime(
  params: StatsPostsOverTimeGetRequest
): Promise<ServerStatsPeriodsResponse> {
  try {
    return await statsApi.statsPostsOverTimeGet(params);
  } catch (error) {
    console.error("get stats posts over time failed:", error);
    throw error;
  }
}

export async function getStatsFilms(
  params: StatsFilmsGetRequest
): Promise<ServerStatsFilmsResponse> {
  try {
    return await statsApi.statsFilmsGet(params);
  } catch (error) {
    console.error("get stats films failed:", error);
    throw error;
  }
}

export async function getStatsCameras(
  params: StatsCamerasGetRequest
): Promise<ServerStatsCamerasResponse> {
  try {
    return await statsApi.statsCamerasGet(params);
  } catch (error) {
    console.error("get stats cameras failed:", error);
    throw error;
  }
}

export async function getStatsColors(
  params: StatsColorsGetRequest
): Promise<ServerStatsColorsResponse> {
  try {
    return await statsApi.statsColorsGet(params);
  } catch (error) {
    console.error("get stats colors failed:", error);
    throw error;
  }
}

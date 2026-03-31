import { ProtectedBox } from '@/app/components/container/protected-box';
import { RapidaIcon } from '@/app/components/Icon/Rapida';
import { RapidaTextIcon } from '@/app/components/Icon/RapidaText';
import { cn } from '@/utils';
import { Microphone, Globe, ChartLine } from '@carbon/icons-react';
import React from 'react';
import { Outlet, Route, Routes, useLocation } from 'react-router-dom';
import {
  OnboardingCreateOrganizationPage,
  OnboardingCreateProjectPage,
} from '@/app/pages/user-onboarding';
import { useWorkspace } from '@/workspace';
import {
  ProgressIndicator,
  ProgressStep,
  Tag,
} from '@carbon/react';

// ── Step definitions ──────────────────────────────────────────────────────────

const STEPS = [
  { path: 'organization', label: 'Create organization', description: 'Set up your workspace', step: 1 },
  { path: 'project', label: 'Create project', description: 'Group your AI resources', step: 2 },
];

// ── Feature highlights ────────────────────────────────────────────────────────

const FEATURES = [
  { icon: Microphone, text: 'Build voice & text AI assistants' },
  { icon: Globe,      text: 'Deploy to any channel in minutes' },
  { icon: ChartLine,  text: 'Monitor quality with real-time analytics' },
];

// ── Layout ────────────────────────────────────────────────────────────────────

function OnboardingLayout() {
  const location = useLocation();
  const workspace = useWorkspace();

  const currentStep =
    STEPS.find(s => location.pathname.includes(s.path))?.step ?? 1;

  return (
    <div className="min-h-[100dvh] flex bg-white dark:bg-gray-900">

      {/* ── Left brand panel ───────────────────────────────────────── */}
      <aside className="hidden lg:flex w-[400px] xl:w-[460px] flex-shrink-0 bg-gray-900 dark:bg-gray-950 flex-col relative overflow-hidden">

        {/* Dot-grid decoration */}
        <div
          className="absolute inset-0 opacity-[0.04] pointer-events-none"
          style={{
            backgroundImage: 'radial-gradient(circle, white 1px, transparent 1px)',
            backgroundSize: '24px 24px',
          }}
        />

        {/* Logo */}
        <div className="relative px-10 pt-10">
          {workspace.logo ? (
            <>
              <img src={workspace.logo.dark} alt={workspace.title} className="h-8" />
            </>
          ) : (
            <div className="flex items-center gap-2 text-white">
              <RapidaIcon className="h-8 w-8" />
              <RapidaTextIcon className="h-6" />
            </div>
          )}
        </div>

        {/* Tagline + feature highlights */}
        <div className="relative flex-1 flex flex-col justify-center px-10 pb-8">
          <p className="text-[10px] font-semibold tracking-[0.16em] uppercase text-gray-400 mb-4">
            Getting started
          </p>
          <h2 className="text-2xl font-light text-white mb-3 leading-snug">
            Build AI assistants that understand and respond like humans.
          </h2>
          <p className="text-sm text-gray-400 leading-relaxed mb-8">
            rapida.ai helps you create, deploy, and monitor voice AI experiences
            — powered by the world's best LLMs and speech engines.
          </p>

          {/* Feature list */}
          <div className="flex flex-col gap-3">
            {FEATURES.map(({ icon: Icon, text }) => (
              <div key={text} className="flex items-center gap-3">
                <div className="w-7 h-7 flex items-center justify-center bg-white/10 shrink-0">
                  <Icon size={16} className="text-gray-300" />
                </div>
                <span className="text-sm text-gray-300 leading-5">{text}</span>
              </div>
            ))}
          </div>
        </div>

      </aside>

      {/* ── Right form panel ───────────────────────────────────────── */}
      <main className="flex-1 flex flex-col min-w-0">

        {/* Mobile header */}
        <div className="lg:hidden flex items-center justify-between px-6 py-4 border-b border-gray-100 dark:border-gray-800">
          <div className="flex items-center gap-2 text-primary">
            <RapidaIcon className="h-7 w-7" />
            <RapidaTextIcon className="h-5" />
          </div>
          <Tag size="sm" type="blue">
            Step {currentStep} of {STEPS.length}
          </Tag>
        </div>

        {/* Form area */}
        <div className="flex-1 flex flex-col items-center justify-center px-6 sm:px-12 py-10">
          <div className="w-full max-w-md">
            {/* Progress indicator — horizontal, above form */}
            <div className="hidden lg:block mb-14">
              <ProgressIndicator currentIndex={currentStep - 1} spaceEqually>
                {STEPS.map(step => (
                  <ProgressStep
                    key={step.path}
                    label={step.label}
                    description={step.description}
                    secondaryLabel={`Step ${step.step}`}
                  />
                ))}
              </ProgressIndicator>
            </div>
            <Outlet />
          </div>
        </div>
      </main>
    </div>
  );
}

// ── Route ─────────────────────────────────────────────────────────────────────

export function OnbaordingRoute() {
  return (
    <Routes>
      <Route
        path="/"
        element={
          <ProtectedBox>
            <OnboardingLayout />
          </ProtectedBox>
        }
      >
        <Route
          key="organization"
          path="organization"
          element={<OnboardingCreateOrganizationPage />}
        />

        <Route
          key="project"
          path="project"
          element={<OnboardingCreateProjectPage />}
        />
      </Route>
    </Routes>
  );
}

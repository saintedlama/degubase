import { createRouter, createWebHistory } from "vue-router";
import { navStart, navDone } from "../foundation/useProgress.js";
import HomeView from "../features/workspace/HomeView.vue";
import WorkspaceView from "../features/workspace/WorkspaceView.vue";
import ApiTokensView from "../features/identity/ApiTokensView.vue";
import UsersView from "../features/identity/UsersView.vue";
import SnapshotsView from "../features/identity/SnapshotsView.vue";
import JobsView from "../features/identity/JobsView.vue";
import TableView from "../features/records/TableView.vue";
import RecordDetailView from "../features/records/RecordDetailView.vue";
import LoginView from "../features/identity/LoginView.vue";
import ScriptsView from "../features/automation/ScriptsView.vue";
import ScriptsCodeTab from "../features/automation/ScriptsCodeTab.vue";
import ScriptsExecutionsTab from "../features/automation/ScriptsExecutionsTab.vue";
import ScriptsReferenceTab from "../features/automation/ScriptsReferenceTab.vue";
import SkillsView from "../features/skills/SkillsView.vue";
import SkillsInstallTab from "../features/skills/SkillsInstallTab.vue";
import SkillsPreviewTab from "../features/skills/SkillsPreviewTab.vue";
import NotFoundView from "../features/workspace/NotFoundView.vue";
import { useAuth } from "../features/identity/useAuth.js";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: "/login", component: LoginView, meta: { public: true } },
    { path: "/", component: HomeView },
    { path: "/workspaces/:workspaceCode", component: WorkspaceView },
    {
      path: "/workspaces/:workspaceCode/settings/tokens",
      component: ApiTokensView,
    },
    {
      path: "/workspaces/:workspaceCode/settings/users",
      component: UsersView,
      meta: { requiresAdmin: true },
    },
    {
      path: "/admin/snapshots",
      component: SnapshotsView,
      meta: { requiresAdmin: true },
    },
    {
      path: "/admin/jobs",
      component: JobsView,
      meta: { requiresAdmin: true },
    },
    {
      path: "/workspaces/:workspaceCode/automations/scripts",
      component: ScriptsView,
      children: [
        { path: ":scriptId/code", component: ScriptsCodeTab },
        { path: ":scriptId/executions", component: ScriptsExecutionsTab },
        { path: ":scriptId/reference", component: ScriptsReferenceTab },
      ],
    },
    {
      path: "/workspaces/:workspaceCode/automations/skills",
      component: SkillsView,
      children: [
        { path: ":skillId/install", component: SkillsInstallTab },
        { path: ":skillId/preview", component: SkillsPreviewTab },
      ],
    },
    {
      path: "/workspaces/:workspaceCode/tables/:tableCode/rows/:rowId",
      component: RecordDetailView,
    },
    {
      path: "/workspaces/:workspaceCode/tables/:tableCode/views/:viewCode",
      component: TableView,
    },
    {
      path: "/:pathMatch(.*)*",
      component: NotFoundView,
      meta: { public: true },
    },
  ],
});

router.beforeEach(async (to) => {
  navStart();
  const { user, checked, authDisabled, check } = useAuth();

  if (!checked.value) {
    await check();
  }

  if (authDisabled.value) {
    if (to.path === "/login") return "/";
    return true;
  }

  if (user.value) {
    if (to.path === "/login") return "/";
    if (to.meta.requiresAdmin && !user.value.is_admin) return "/";
    return true;
  }

  if (to.meta.public) return true;
  return "/login";
});

router.afterEach(() => navDone());

export default router;

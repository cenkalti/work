-- work.lua: WezTerm event handler for the agent dashboard.
--
-- Usage in your wezterm.lua:
--
--   package.path = '/Users/you/projects/work/wezterm/?.lua;' .. package.path
--   require('work').setup()
--
-- Then bind a key in your keys file:
--
--   { mods = 'SUPER', key = 'd', action = wezterm.action.EmitEvent('work-toggle-dashboard') }
--
-- 'work-toggle-dashboard' toggles between the dashboard and the previously-focused
-- pane. If no dashboard has been spawned (or its pane is gone), spawns a new
-- window running `agent dash` and remembers it.

local wezterm = require('wezterm')

local M = {}

local agent_bin = os.getenv('HOME') .. '/go/bin/agent'

-- Module-level state, kept in WezTerm's Lua memory:
--   dashboard_pane_id: pane id of the spawned dashboard, or nil if not running.
--   last_focus:        the most recently focused non-dashboard pane id.
---@type integer?
local dashboard_pane_id = nil
---@type integer?
local last_focus = nil

-- safe_get_pane returns the mux pane or nil. wezterm.mux.get_pane throws on
-- missing panes, so we pcall it.
---@param pane_id integer
local function safe_get_pane(pane_id)
    local ok, p = pcall(wezterm.mux.get_pane, pane_id)
    if not ok then
        return nil
    end
    return p
end

-- activate_pane focuses the pane within its tab and raises the GUI window.
---@param pane_id integer
---@return boolean
local function activate_pane(pane_id)
    local target = safe_get_pane(pane_id)
    if not target then
        return false
    end
    target:activate()
    local mw = target:window()
    if mw then
        local gw = mw:gui_window()
        if gw then
            gw:focus()
        end
    end
    return true
end

function M.setup()
    -- Maximize the window when an agent pane signals via the agent_maximize
    -- user var (set by the dashboard when jumping into an agent).
    wezterm.on('user-var-changed', function(window, _, name, _)
        if name ~= 'agent_maximize' then
            return
        end
        window:maximize()
    end)

    wezterm.on('work-toggle-dashboard', function(_, pane)
        local current = pane:pane_id()

        -- Drop a stale dashboard reference (window was closed externally).
        if dashboard_pane_id and not safe_get_pane(dashboard_pane_id) then
            dashboard_pane_id = nil
        end

        if dashboard_pane_id and current == dashboard_pane_id then
            if last_focus then
                activate_pane(last_focus)
            end
            return
        end

        last_focus = current
        if dashboard_pane_id and activate_pane(dashboard_pane_id) then
            return
        end

        -- Spawn a new dashboard window and remember its pane id.
        local _, new_pane, new_window = wezterm.mux.spawn_window({
            args = { agent_bin, 'dash' },
        })
        if new_pane then
            dashboard_pane_id = new_pane:pane_id()
        end
        if new_window then
            local gw = new_window:gui_window()
            if gw then
                gw:maximize()
                gw:focus()
            end
        end
    end)
end

return M

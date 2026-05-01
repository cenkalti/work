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
-- pane. If no dashboard is recorded or its pane is gone, a new window is spawned
-- running `agent dash`.
--
-- Inside the dashboard TUI:
--   j / k        move cursor
--   enter        jump to the highlighted agent
--   1..9         jump to that slot's agent (no modifier)
--   alt+1..9     assign the highlighted agent to slot N
--   alt+0        unassign the highlighted agent's slot
--   q            quit

local wezterm = require('wezterm')

local M = {}

local home = os.getenv('HOME')
local dashboard_path = home .. '/.work/dashboard.json'
local agent_bin = home .. '/go/bin/agent'

-- last_focus is the most recently focused non-dashboard pane id (integer).
-- Used to toggle back when the event fires from the dashboard pane.
---@type integer?
local last_focus = nil

local function read_json(path)
    local f = io.open(path, 'r')
    if not f then
        return nil
    end
    local body = f:read('*a')
    f:close()
    local ok, parsed = pcall(wezterm.json_parse, body)
    if not ok then
        return nil
    end
    return parsed
end

-- activate_pane focuses the pane within its tab and raises the GUI window.
---@param pane_id integer
---@return boolean
local function activate_pane(pane_id)
    local target = wezterm.mux.get_pane(pane_id)
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
    wezterm.on('work-toggle-dashboard', function(_, pane)
        local d = read_json(dashboard_path)
        local current = pane:pane_id()
        local dash_pane = d and tonumber(d.pane_id) or nil

        if dash_pane and current == dash_pane then
            if last_focus then
                activate_pane(last_focus)
            end
            return
        end

        last_focus = current
        if dash_pane and activate_pane(dash_pane) then
            return
        end

        wezterm.mux.spawn_window({
            args = { agent_bin, 'dash' },
        })
    end)
end

return M

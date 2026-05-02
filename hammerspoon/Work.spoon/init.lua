local obj = {}
obj.__index = obj
obj.name = 'Work'
obj.version = '0.1'
obj.author = 'Cenk Alti'
obj.license = 'MIT'

obj.pollInterval = 2
obj.agentBin = os.getenv('HOME') .. '/go/bin/agent'
obj.agentsDir = os.getenv('HOME') .. '/.work/agents'

function obj:init()
    self.idleImage = hs.image.imageFromPath(hs.spoons.resourcePath('idle.png')):template(true)
    self.attentionImage = hs.image.imageFromPath(hs.spoons.resourcePath('attention.png'))
    self.menubar = nil
    self.timer = nil
    self.attention = nil
end

function obj:start()
    if self.menubar then return self end
    self.menubar = hs.menubar.new()
    self.menubar:setClickCallback(function() self:focusDashboard() end)
    self.attention = nil
    self:tick()
    self.timer = hs.timer.doEvery(self.pollInterval, function() self:tick() end)
    return self
end

function obj:tick()
    local attention = false
    local ok, iter, dirObj = pcall(hs.fs.dir, self.agentsDir)
    if not ok or not iter then return end
    for name in iter, dirObj do
        if name:sub(-5) == '.json' and name:sub(1, 1) ~= '.' then
            local rec = hs.json.read(self.agentsDir .. '/' .. name)
            if rec and not rec.archived then
                if rec.status == 'awaiting_input' or (rec.notification_count or 0) > 0 then
                    attention = true
                    break
                end
            end
        end
    end
    if attention == self.attention then return end
    self.attention = attention
    if attention then
        self.menubar:setIcon(self.attentionImage, false)
    else
        self.menubar:setIcon(self.idleImage, true)
    end
end

function obj:focusDashboard()
    hs.task.new(self.agentBin, nil, { 'dash-focus' }):start()
end

function obj:stop()
    if self.timer then
        self.timer:stop()
        self.timer = nil
    end
    if self.menubar then
        self.menubar:delete()
        self.menubar = nil
    end
    self.attention = nil
    return self
end

return obj

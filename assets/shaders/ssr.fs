#version 330

// DEBUG: Simple passthrough to test render-to-texture pipeline

in vec2 fragTexCoord;

uniform sampler2D texture0;      // Main rendered scene (bound by DrawTextureRec)

out vec4 finalColor;

void main() {
    // Just pass through the color buffer for now
    finalColor = texture(texture0, fragTexCoord);
}

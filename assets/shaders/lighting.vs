#version 330

in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;
uniform mat4 matModel;
uniform mat4 matNormal;
uniform mat4 matLightVP;  // Light view-projection matrix for shadowmapping

out vec3 fragPosition;
out vec2 fragTexCoord;
out vec4 fragColor;
out vec3 fragNormal;
out vec2 fragShadowTexCoord;
out float fragShadowDepth;

void main()
{
    // World space position
    vec4 worldPos = matModel * vec4(vertexPosition, 1.0);
    fragPosition = worldPos.xyz;

    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;
    fragNormal = normalize(vec3(matNormal * vec4(vertexNormal, 1.0)));

    // Calculate shadow coordinates
    vec4 shadowClipPos = matLightVP * worldPos;
    // Convert from clip space (-1 to 1) to texture space (0 to 1)
    // BOTH xy and depth need this conversion to match the depth buffer
    fragShadowTexCoord = (shadowClipPos.xy / shadowClipPos.w) * 0.5 + 0.5;
    fragShadowDepth = (shadowClipPos.z / shadowClipPos.w) * 0.5 + 0.5;

    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
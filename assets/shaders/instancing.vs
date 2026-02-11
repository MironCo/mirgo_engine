#version 330

// Input vertex attributes
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

// Instance transform (mat4 split across 4 vec4 attributes)
in mat4 instanceTransform;

uniform mat4 mvp;
uniform mat4 matLightVP;

out vec3 fragPosition;
out vec2 fragTexCoord;
out vec4 fragColor;
out vec3 fragNormal;
out vec3 fragTangent;
out vec3 fragBitangent;
out vec2 fragShadowTexCoord;
out float fragShadowDepth;

void main()
{
    // World space position using instance transform
    vec4 worldPos = instanceTransform * vec4(vertexPosition, 1.0);
    fragPosition = worldPos.xyz;

    fragTexCoord = vertexTexCoord;
    fragColor = vertexColor;

    // Calculate normal matrix from instance transform (inverse transpose of upper 3x3)
    mat3 normalMatrix = mat3(instanceTransform);
    fragNormal = normalize(normalMatrix * vertexNormal);

    // No tangent/bitangent for instanced objects (no normal maps)
    fragTangent = vec3(1.0, 0.0, 0.0);
    fragBitangent = vec3(0.0, 1.0, 0.0);

    // Calculate shadow coordinates
    vec4 shadowClipPos = matLightVP * worldPos;
    fragShadowTexCoord = (shadowClipPos.xy / shadowClipPos.w) * 0.5 + 0.5;
    fragShadowDepth = (shadowClipPos.z / shadowClipPos.w) * 0.5 + 0.5;

    gl_Position = mvp * worldPos;
}